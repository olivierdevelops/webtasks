// Package rest holds the HTTP "view" — dumb handlers that decode requests,
// call use cases, encode responses. Each handler declares (locally) the
// use-case protocol it consumes; the orchestrator supplies values satisfying
// those protocols.
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"webtasks/internal/domain"
	"webtasks/internal/features"
)

// UseCases groups the consumer-owned protocols this io layer needs.
type UseCases struct {
	RunTask    func(name string, vals domain.InputValues) (domain.Output, error)
	StreamTask func(ctx context.Context, name string, vals domain.InputValues, events features.EventPublisher) (domain.Output, error)
	ListTasks  func() []domain.TaskDef
	Health     func() map[string]any
	PoolStatus func() map[string]features.PoolStatus
}

// Register wires the REST endpoints onto the given chi router.
func Register(r chi.Router, uc UseCases) {
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, uc.Health())
	})

	r.Get("/tasks", func(w http.ResponseWriter, _ *http.Request) {
		tasks := uc.ListTasks()
		out := make([]map[string]any, 0, len(tasks))
		for _, t := range tasks {
			tt := make([]string, 0, len(t.Transports))
			for _, x := range t.Transports {
				tt = append(tt, string(x))
			}
			schema := make(map[string]any, len(t.InputSchema))
			for k, f := range t.InputSchema {
				schema[k] = map[string]any{
					"type": f.Type, "required": f.Required, "default": f.DefaultValue, "doc": f.Doc,
				}
			}
			out = append(out, map[string]any{
				"name":       t.Name,
				"poolTag":    string(t.PoolTag),
				"transports": tt,
				"timeoutMs":  t.TimeoutMs,
				"input":      schema,
			})
		}
		writeJSON(w, http.StatusOK, out)
	})

	// POST /tasks/<name…> — chi captures the wildcard as `*`. When the
	// caller asks for `text/event-stream`, events are streamed; otherwise
	// the response is one synchronous JSON object.
	r.Post("/tasks/*", func(w http.ResponseWriter, req *http.Request) {
		name := strings.TrimPrefix(chi.URLParam(req, "*"), "/")
		if name == "" {
			writeJSON(w, http.StatusBadRequest, errBody("MISSING_NAME", "task name is required"))
			return
		}
		vals := domain.InputValues{}
		if req.ContentLength != 0 {
			if err := json.NewDecoder(req.Body).Decode(&vals); err != nil {
				writeJSON(w, http.StatusBadRequest, errBody("BAD_BODY", err.Error()))
				return
			}
		}
		if wantsSSE(req) {
			streamSSE(w, req, uc, name, vals)
			return
		}
		out, err := uc.RunTask(name, vals)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("EXECUTION_FAILED", err.Error()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": resultData(out)})
	})
}

// resultData unwraps a task's response payload: when a `return` action set the
// reserved `__result__` key, that value alone is the payload; otherwise the
// full Output map is returned (back-compatible).
func resultData(out domain.Output) any {
	if v, ok := out["__result__"]; ok {
		return v
	}
	return out
}

func wantsSSE(req *http.Request) bool {
	return strings.Contains(strings.ToLower(req.Header.Get("Accept")), "text/event-stream")
}

// streamSSE wires an EventPublisher whose Emit method writes a server-sent
// event for every progress message, then a terminal `done` (with the final
// Output) or `error` event before the connection closes.
func streamSSE(w http.ResponseWriter, req *http.Request, uc UseCases, name string, vals domain.InputValues) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, errBody("NO_FLUSHER", "response writer cannot stream"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := req.Context()
	events := features.EventPublisher{Emit: func(e domain.Event) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		writeSSE(w, flusher, e.Kind, map[string]any{"text": e.Text, "data": e.Data})
	}}

	done := make(chan struct{})
	var (
		out domain.Output
		err error
	)
	go func() {
		defer close(done)
		out, err = uc.StreamTask(ctx, name, vals, events)
	}()

	// Heartbeat so intermediaries don't drop the connection.
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-done:
			if err != nil {
				writeSSE(w, flusher, "error", map[string]any{"message": err.Error()})
			} else {
				writeSSE(w, flusher, "done", map[string]any{"ok": true, "data": resultData(out)})
			}
			return
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any) {
	body, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, body)
	flusher.Flush()
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func errBody(code, msg string) map[string]any {
	return map[string]any{
		"ok":    false,
		"error": map[string]any{"code": code, "message": msg},
	}
}
