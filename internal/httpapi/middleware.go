// Package httpapi is the contract-driven transport and transaction boundary.
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/google/uuid"
)

var errNotImplemented = errors.New("operation not implemented")

type requestMeta struct {
	route, trace string
	raw          []byte
	scope        string
}
type metaKey struct{}

// RawBody is an immutable copy of the bounded original bytes. Billing must verify
// the provider signature over these bytes before persisting or acknowledging.
func RawBody(ctx context.Context) []byte {
	if m, ok := ctx.Value(metaKey{}).(*requestMeta); ok {
		return bytes.Clone(m.raw)
	}
	return nil
}
func ClientTrace(ctx context.Context) string {
	if m, ok := ctx.Value(metaKey{}).(*requestMeta); ok {
		return m.trace
	}
	return ""
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w statusWriter) WriteHeader(int) { w.ResponseWriter.WriteHeader(w.status) }
func WriteError(w http.ResponseWriter, r *http.Request, e error) {
	if errors.Is(e, errNotImplemented) {
		apierr.Write(statusWriter{w, 501}, r, apierr.New(apierr.InternalError, "Operation is not implemented."))
		return
	}
	apierr.Write(w, r, e)
}

type bufferWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (w *bufferWriter) Header() http.Header { return w.header }
func (w *bufferWriter) WriteHeader(n int) {
	if w.status == 0 {
		w.status = n
	}
}
func (w *bufferWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	if w.body.Len()+len(b) > 32<<20 {
		return 0, errors.New("response too large")
	}
	return w.body.Write(b)
}

// Transport is deliberately allowlist-only logging. No request/response headers,
// URLs, bodies, panic values or error strings enter access logs (§16.2, SEC02).
func Transport(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := uuid.Must(uuid.NewV7()).String()
		trace := r.Header.Get("X-Request-ID")
		if len(trace) > 256 {
			trace = ""
		}
		meta := &requestMeta{route: "unmatched", trace: trace}
		ctx := context.WithValue(apierr.ContextWithRequestID(r.Context(), id), metaKey{}, meta)
		r = r.WithContext(ctx)
		b := &bufferWriter{header: make(http.Header)}
		func() {
			defer func() {
				if recover() != nil {
					b = &bufferWriter{header: make(http.Header)}
					apierr.Write(b, r, errors.New("panic"))
				}
			}()
			next.ServeHTTP(b, r)
		}()
		if b.status == 0 {
			b.status = 200
		}
		b.header.Set("X-Request-ID", id)
		b.header.Set("X-Content-Type-Options", "nosniff")
		b.header.Set("Referrer-Policy", "no-referrer")
		b.header.Set("X-Frame-Options", "DENY")
		b.header.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		b.header.Set("Strict-Transport-Security", "max-age=31536000")
		for k, v := range b.header {
			w.Header()[k] = v
		}
		w.WriteHeader(b.status)
		if r.Method != "HEAD" {
			_, _ = w.Write(b.body.Bytes())
		}
		if log != nil {
			log.InfoContext(ctx, "http request", "request_id", id, "route", meta.route, "status", b.status, "duration_ms", time.Since(start).Milliseconds())
		}
	})
}

type bucketState struct {
	tokens float64
	at     time.Time
}
type limiter struct {
	mu      sync.Mutex
	entries map[string]bucketState
	max     int
}

func (l *limiter) allow(key string, b config.Bucket) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	s, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= l.max {
			for k, v := range l.entries {
				if now.Sub(v.at) > time.Hour {
					delete(l.entries, k)
				}
			}
			if len(l.entries) >= l.max {
				return time.Second
			}
		}
		s = bucketState{float64(b.Burst), now}
	}
	s.tokens = math.Min(float64(b.Burst), s.tokens+now.Sub(s.at).Seconds()*b.PerSecond)
	s.at = now
	delay := time.Duration(0)
	if s.tokens < 1 {
		delay = time.Duration(math.Ceil((1 - s.tokens) / b.PerSecond * float64(time.Second)))
	} else {
		s.tokens--
	}
	l.entries[key] = s
	return delay
}

type Options struct {
	Config        config.Config
	Sessions      auth.SessionAuthenticator
	Agents        auth.AgentAuthenticator
	AgentListener bool
	Commands      *Commands
}

// New validates the immutable contract once; security metadata comes from tags,
// security requirements and x-fvpn-auth, including enrollment's explicit exception.
func New(c *Composite, o Options) (http.Handler, error) {
	doc, e := gen.GetSwagger()
	if e != nil {
		return nil, e
	}
	doc.Servers = nil
	router, e := legacy.NewRouter(doc)
	if e != nil {
		return nil, e
	}
	lim := &limiter{entries: map[string]bucketState{}, max: o.Config.MaxRatePrincipals}
	if lim.max < 1 {
		return nil, errors.New("rate capacity required")
	}
	var mids []gen.StrictMiddlewareFunc
	if o.Commands != nil {
		mids = append(mids, o.Commands.middleware(c))
	} else {
		mids = append(mids, requireCommandBoundary)
	}
	strict := gen.NewStrictHandlerWithOptions(c, mids, gen.StrictHTTPServerOptions{RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
		apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Malformed request."))
	}, ResponseErrorHandlerFunc: WriteError})
	handler := gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{BaseURL: "/api/v1", ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
		apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Malformed request."))
	}})
	gates := map[string]chan struct{}{}
	for _, class := range []string{"browser", "agent", "heartbeat", "lease", "usage", "anonymous"} {
		gates[class] = make(chan struct{}, o.Config.MaxConcurrent)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		route, params, e := findRoute(router, r)
		if e != nil {
			code := apierr.ResourceNotFound
			if errors.Is(e, routers.ErrMethodNotAllowed) {
				code = apierr.MethodNotAllowed
			}
			apierr.Write(w, r, apierr.New(code, "Route not available."))
			return
		}
		op := route.Operation
		if len(op.Tags) != 1 {
			apierr.Write(w, r, errors.New("invalid route security"))
			return
		}
		tag := op.Tags[0]
		mtls := op.Extensions["x-fvpn-auth"] == "agent-mtls"
		// Extension values may be json.RawMessage depending on the loader version.
		if v, ok := op.Extensions["x-fvpn-auth"].(json.RawMessage); ok {
			mtls = string(v) == `"agent-mtls"`
		}
		if o.AgentListener != mtls {
			apierr.Write(w, r, apierr.New(apierr.Unauthenticated, "Use the authenticated listener."))
			return
		}
		m, _ := r.Context().Value(metaKey{}).(*requestMeta)
		if m == nil {
			m = &requestMeta{}
			r = r.WithContext(context.WithValue(r.Context(), metaKey{}, m))
		}
		m.route = r.Method + " /api/v1" + route.Path
		var p auth.Principal
		if tag == "admin" || tag == "member" {
			if o.Sessions == nil {
				e = apierr.New(apierr.Unauthenticated, "Sign in required.")
			} else {
				p, e = o.Sessions.AuthenticateSession(r)
			}
			if e == nil && (string(p.Role) != tag || p.ID == uuid.Nil) {
				e = apierr.New(apierr.Forbidden, "Role is not allowed.")
			}
			if e == nil && tag == "admin" && !p.MFAVerified {
				e = apierr.New(apierr.Forbidden, "Administrator MFA required.")
			}
		} else if mtls {
			if o.Agents == nil || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 {
				e = apierr.New(apierr.Unauthenticated, "Verified client certificate required.")
			} else {
				p, e = o.Agents.AuthenticateAgent(r.Context(), r.TLS.VerifiedChains[0][0])
				if e == nil && (p.Role != auth.Agent || p.NodeID == uuid.Nil || p.IncarnationID == uuid.Nil) {
					e = apierr.New(apierr.Unauthenticated, "Invalid machine identity.")
				}
			}
		}
		if e != nil {
			apierr.Write(w, r, e)
			return
		}
		r = r.WithContext(auth.WithPrincipal(r.Context(), p))
		class, bounds, bodyLimit := classify(op, o.Config)
		key := p.Scope()
		if p.Role == "" {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			key = "anonymous:" + host
		}
		if delay := lim.allow(class+":"+key, bounds); delay > 0 {
			apierr.Write(w, r, apierr.New(apierr.RateLimited, "Request rate exceeded.").WithRetryAfter(delay))
			return
		}
		select {
		case gates[class] <- struct{}{}:
			defer func() { <-gates[class] }()
		default:
			apierr.Write(w, r, apierr.New(apierr.RateLimited, "Request concurrency exceeded.").WithRetryAfter(time.Second))
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && (p.Role == auth.Admin || p.Role == auth.Member) {
			if e = auth.CheckCSRF(r, p, o.Config.PublicOrigin); e != nil {
				apierr.Write(w, r, e)
				return
			}
			if p.Role == auth.Admin && (p.AuthenticatedAt.IsZero() || time.Since(p.AuthenticatedAt) > time.Duration(o.Config.RecentAuthSeconds)*time.Second || p.AuthenticatedAt.After(time.Now().Add(time.Minute))) {
				apierr.Write(w, r, apierr.New(apierr.Forbidden, "Recent administrator authentication required."))
				return
			}
		}
		for _, par := range op.Parameters {
			if par.Value != nil && par.Value.In == "header" && par.Value.Name == "If-Match" && par.Value.Required && r.Header.Get("If-Match") == "" {
				apierr.Write(w, r, apierr.New(apierr.PreconditionRequired, "If-Match is required."))
				return
			}
		}
		raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, bodyLimit))
		if e != nil {
			var size *http.MaxBytesError
			if errors.As(e, &size) {
				apierr.Write(w, r, apierr.New(apierr.PayloadTooLarge, "Request body exceeds the configured limit."))
			} else {
				apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Cannot read request body."))
			}
			return
		}
		m.raw = raw
		if len(raw) > 0 && tag != "webhook" {
			typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || typ != "application/json" {
				apierr.Write(w, r, apierr.New(apierr.UnsupportedMediaType, "JSON content required."))
				return
			}
			if _, err = canonjson.Canonicalize(raw); err != nil {
				apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Invalid JSON document."))
				return
			}
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		validation := &openapi3filter.RequestValidationInput{Request: r, PathParams: params, Route: route, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc, SkipSettingDefaults: true, ExcludeRequestBody: tag == "webhook", RejectWhenRequestBodyNotSpecified: true}}
		if e = openapi3filter.ValidateRequest(r.Context(), validation); e != nil {
			apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Request does not match the API schema."))
			return
		}
		if mtls && len(raw) > 0 {
			var body any
			_ = json.Unmarshal(raw, &body)
			if !matchingIdentity(body, p) {
				apierr.Write(w, r, apierr.New(apierr.IncarnationMismatch, "Request identity differs from the certificate."))
				return
			}
		}
		// Concrete path parameters are part of the scope. Reusing a key on another
		// resource must never replay the first resource's command (§14.5).
		m.scope = r.Method + " " + route.Path
		for _, name := range parameterNames(route.Path) {
			m.scope += "|" + name + "=" + params[name]
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		handler.ServeHTTP(w, r)
	}), nil
}
func findRoute(router routers.Router, r *http.Request) (*routers.Route, map[string]string, error) {
	clone := r.Clone(r.Context())
	u := *r.URL
	clone.URL = &u
	clone.URL.Path = strings.TrimPrefix(r.URL.Path, "/api/v1")
	return router.FindRoute(clone)
}
func parameterNames(path string) []string {
	var names []string
	for _, p := range strings.Split(path, "/") {
		if strings.HasPrefix(p, "{") {
			names = append(names, strings.Trim(p, "{}"))
		}
	}
	return names
}
func classify(op *openapi3.Operation, c config.Config) (string, config.Bucket, int64) {
	switch op.OperationID {
	case "submitObservation":
		return "heartbeat", c.HeartbeatRate, c.AgentBodyBytes
	case "renewServingLease":
		return "lease", c.LeaseRate, c.AgentBodyBytes
	case "submitUsageBatch":
		return "usage", c.UsageRate, c.UsageBodyBytes
	case "enrollNode":
		return "anonymous", c.AnonymousRate, c.AgentBodyBytes
	case "receiveProviderWebhook":
		return "anonymous", c.AnonymousRate, c.WebhookBodyBytes
	}
	if op.Tags[0] == "agent" {
		return "agent", c.AgentRate, c.AgentBodyBytes
	}
	return "browser", c.BrowserRate, c.BrowserBodyBytes
}
func matchingIdentity(v any, p auth.Principal) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, v := range x {
			if k == "node_id" && v != p.NodeID.String() {
				return false
			}
			if (k == "node_incarnation_id" || k == "incarnation_id") && v != p.IncarnationID.String() {
				return false
			}
			if !matchingIdentity(v, p) {
				return false
			}
		}
	case []any:
		for _, v := range x {
			if !matchingIdentity(v, p) {
				return false
			}
		}
	}
	return true
}
