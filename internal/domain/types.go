// Package domain holds the pure data shapes of the system. No behaviour
// beyond simple constructors / transforms. Mirrors the Java domain package.
package domain

import "time"

// PoolTag names a window pool (e.g. "concio", "default").
type PoolTag string

// WindowID is the pool-assigned identifier of a single Chrome window.
type WindowID string

// JobID identifies an async job.
type JobID string

// JobStatus enumerates possible job life-cycle states.
type JobStatus string

const (
	JobQueued    JobStatus = "QUEUED"
	JobRunning   JobStatus = "RUNNING"
	JobDone      JobStatus = "DONE"
	JobError     JobStatus = "ERROR"
	JobCancelled JobStatus = "CANCELLED"
)

// Transport names a way of invoking a task over HTTP.
type Transport string

const (
	TransportREST      Transport = "rest"
	TransportSSE       Transport = "sse"
	TransportWebSocket Transport = "websocket"
	TransportAsync     Transport = "async"
)

// InputField describes one declared parameter of a task.
type InputField struct {
	Name         string `yaml:"-"`
	Type         string `yaml:"type"`
	Required     bool   `yaml:"required"`
	DefaultValue any    `yaml:"default"`
	Doc          string `yaml:"doc"`
}

// InputValues is the bag of caller-supplied input values for a task run.
type InputValues map[string]any

// Output accumulates values produced by `as:` steps during execution.
type Output map[string]any

// Command is one step in a task flow.
type Command struct {
	Run    string `yaml:"run"`
	Status string `yaml:"status,omitempty"`
	As     string `yaml:"as,omitempty"`
	// Record, when true, screencasts just this step's execution to a GIF (a
	// debug aid for diagnosing why a step fails). The recording is always
	// kept and its path is reported via a `recording` event.
	Record   bool           `yaml:"record,omitempty"`
	Params   map[string]any `yaml:"params,omitempty"`
	Children []Command      `yaml:"do,omitempty"`
}

// TaskDef is a single registered task: its name, allowed transports, input
// shape, flow, and limits.
type TaskDef struct {
	Name        string                `yaml:"name"`
	PoolTag     PoolTag               `yaml:"poolTag"`
	Transports  []Transport           `yaml:"transports"`
	InputSchema map[string]InputField `yaml:"input"`
	Flow        []Command             `yaml:"flow"`
	TimeoutMs   int64                 `yaml:"timeoutMs"`
	// SetupTask, when non-empty, names another registered task whose flow runs
	// inside the same window lease *before* this task's flow. The referenced
	// task must be idempotent (a no-op when its post-condition already holds).
	// Useful for "ensure logged in" preludes.
	SetupTask string `yaml:"setupTask,omitempty"`
}

func (t TaskDef) Timeout() time.Duration {
	if t.TimeoutMs <= 0 {
		return 60 * time.Second
	}
	return time.Duration(t.TimeoutMs) * time.Millisecond
}

// Event is a progress event emitted during a run (delivered over SSE/WS,
// dropped on the floor under sync REST).
type Event struct {
	Kind string         `json:"kind"`
	Text string         `json:"text"`
	Data map[string]any `json:"data,omitempty"`
}

// StaticMount declaratively maps a URL prefix to a local directory.
type StaticMount struct {
	Prefix    string `yaml:"prefix"`
	Dir       string `yaml:"dir"`
	List      bool   `yaml:"list"`
	Serve     bool   `yaml:"serve"`
	Recursive bool   `yaml:"recursive"`
}

// SecretDecl declares a runtime value the server expects (read at startup,
// stored as a JVM-equivalent property keyed by Name).
type SecretDecl struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Required     bool     `yaml:"required"`
	Sensitive    bool     `yaml:"sensitive"`
	DefaultValue string   `yaml:"default"`
	Sources      []string `yaml:"sources"`
}

// ExtractFieldSpec describes one field within an extract spec.
type ExtractFieldSpec struct {
	Kind       string `yaml:"kind"`
	Selector   string `yaml:"selector"`
	AttrName   string `yaml:"name"`
	Transform  string `yaml:"transform,omitempty"`
	ConstValue any    `yaml:"value,omitempty"`
}

// ExtractSpec is what the `extract` command consumes.
type ExtractSpec struct {
	Selector string                      `yaml:"selector"`
	Repeat   bool                        `yaml:"repeat"`
	Fields   map[string]ExtractFieldSpec `yaml:"fields"`
}
