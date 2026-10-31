// Package granttrace is TEST ONLY control composition and original wire evidence.
package granttrace

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

const address = "127.0.0.1:19090"

type Event struct {
	RequestWire    string          `json:"request_wire,omitempty"`
	ResponseWire   string          `json:"response_wire,omitempty"`
	RequestSHA256  string          `json:"request_sha256,omitempty"`
	ResponseSHA256 string          `json:"response_sha256,omitempty"`
	Started        time.Time       `json:"started"`
	Completed      time.Time       `json:"completed"`
	Kind           string          `json:"kind"`
	Ready          *bool           `json:"ready,omitempty"`
	Status         int             `json:"http_status,omitempty"`
	RetryAfter     string          `json:"retry_after,omitempty"`
	ETag           string          `json:"etag,omitempty"`
	Request        json.RawMessage `json:"request,omitempty"`
	Response       json.RawMessage `json:"response,omitempty"`
}
type Recorder struct {
	mu     sync.Mutex
	events []Event
}

func (r *Recorder) add(e Event) { r.mu.Lock(); defer r.mu.Unlock(); r.events = append(r.events, e) }

type writer struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *writer) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}
func (w *writer) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, e := w.ResponseWriter.Write(p)
	if w.body.Len()+n <= 2<<20 {
		w.body.Write(p[:n])
	}
	return n, e
}
func (r *Recorder) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		kind := ""
		switch q.URL.Path {
		case "/api/v1/agent/lease-renewals":
			kind = "lease"
		case "/api/v1/agent/observations":
			kind = "observation"
		case "/api/v1/agent/desired-state":
			kind = "desired"
		}
		if kind == "" {
			next.ServeHTTP(w, q)
			return
		}
		e := Event{Kind: kind, Started: time.Now().UTC()}
		if q.Body != nil {
			b, err := io.ReadAll(io.LimitReader(q.Body, 2<<20))
			if err != nil {
				http.Error(w, "fixture read failed", 500)
				return
			}
			q.Body = io.NopCloser(bytes.NewReader(b))
			if json.Valid(b) {
				e.Request = b
				e.RequestWire = string(b)
				sum := sha256.Sum256(b)
				e.RequestSHA256 = hex.EncodeToString(sum[:])
			}
		}
		capture := &writer{ResponseWriter: w}
		next.ServeHTTP(capture, q)
		e.Completed = time.Now().UTC()
		e.Status = capture.status
		e.RetryAfter = w.Header().Get("Retry-After")
		e.ETag = w.Header().Get("ETag")
		if json.Valid(capture.body.Bytes()) {
			e.Response = append([]byte(nil), capture.body.Bytes()...)
			e.ResponseWire = capture.body.String()
			sum := sha256.Sum256(capture.body.Bytes())
			e.ResponseSHA256 = hex.EncodeToString(sum[:])
		}
		r.add(e)
	})
}

// Listen binds only inside the isolated control container. No product route,
// environment flag, authentication bypass or transport fault is installed.
func (r *Recorder) Listen(set func(bool)) (func(), error) {
	ln, err := net.Listen("tcp4", address)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /member-grants", func(w http.ResponseWriter, q *http.Request) {
		var in struct {
			Ready *bool `json:"ready"`
		}
		if json.NewDecoder(io.LimitReader(q.Body, 128)).Decode(&in) != nil || in.Ready == nil {
			http.Error(w, "ready required", 400)
			return
		}
		e := Event{Kind: "flip", Started: time.Now().UTC(), Ready: in.Ready}
		set(*in.Ready)
		e.Completed = time.Now().UTC()
		r.add(e)
		_ = json.NewEncoder(w).Encode(e)
	})
	mux.HandleFunc("GET /evidence", func(w http.ResponseWriter, _ *http.Request) {
		r.mu.Lock()
		defer r.mu.Unlock()
		_ = json.NewEncoder(w).Encode(r.events)
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	return func() { _ = srv.Close() }, nil
}
func Command(command string, out io.Writer) error {
	method, path, body := "GET", "/evidence", ""
	switch command {
	case "true", "false":
		method, path, body = "POST", "/member-grants", `{"ready":`+command+`}`
	case "evidence":
	default:
		return fmt.Errorf("unknown fixture command")
	}
	req, err := http.NewRequest(method, "http://"+address+path, bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fixture control unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("fixture HTTP %d", res.StatusCode)
	}
	_, err = io.Copy(out, res.Body)
	return err
}
