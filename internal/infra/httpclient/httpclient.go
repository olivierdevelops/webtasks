// Package httpclient is a thin outbound-HTTP adapter backing the engine's
// `http-request` action. It wraps net/http and knows nothing of the system.
package httpclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

// Response is the raw result of an HTTP call.
type Response struct {
	Status  int
	Headers map[string]string
	Body    string
}

// Client performs outbound HTTP requests.
type Client struct{}

// Do performs one HTTP request. A zero `timeoutMs` defaults to 30s; the call
// is also bound to `ctx` (the task's deadline).
func (Client) Do(ctx context.Context, method, url string, headers map[string]string,
	body []byte, timeoutMs int64, followRedirects bool) (Response, error) {

	if method == "" {
		method = http.MethodGet
	}
	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(reqCtx, method, url, rdr)
	if err != nil {
		return Response{}, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	if !followRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	rh := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		rh[k] = resp.Header.Get(k)
	}
	return Response{Status: resp.StatusCode, Headers: rh, Body: string(data)}, nil
}
