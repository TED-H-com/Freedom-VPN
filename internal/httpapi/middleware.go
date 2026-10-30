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
	"strconv"
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
	"github.com/prometheus/client_golang/prometheus"
)

var errNotImplemented = errors.New("operation not implemented")

type requestMeta struct {
	route, trace string
	raw          []byte
	scope        string
	logger       *slog.Logger
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

func WriteError(w http.ResponseWriter, r *http.Request, e error) {
	if errors.Is(e, errNotImplemented) {
		apierr.Write(w, r, apierr.New(apierr.NotImplemented, "Operation is not implemented."))
		return
	}
	var typed *apierr.Error
	if errors.As(e, &typed) && typed != nil {
		if spec, ok := apierr.Lookup(typed.Code); ok && spec.HTTPStatus == 0 {
			// Revision 3: report-only codes are payload evidence, never HTTP errors.
			// Log only the known enum and central ID, never the error text/details.
			ctx := context.Background()
			logger := slog.Default()
			if r != nil {
				ctx = r.Context()
				if m, ok := ctx.Value(metaKey{}).(*requestMeta); ok && m.logger != nil {
					logger = m.logger
				}
			}
			requestID, _ := apierr.RequestIDFromContext(ctx)
			logger.ErrorContext(ctx, "report-only error rendered as INTERNAL_ERROR", "request_id", requestID, "reported_code", typed.Code)
		}
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
	return TransportWithMetrics(log, next, nil)
}

// TransportWithMetrics counts final responses, including recovered panics, using
// only bounded route templates and status codes (§16.4, SEC02).
func TransportWithMetrics(log *slog.Logger, next http.Handler, requests *prometheus.CounterVec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = apierr.TrackResponse(w)
		start := time.Now()
		id := uuid.Must(uuid.NewV7()).String()
		trace := r.Header.Get("X-Request-ID")
		if len(trace) > 256 {
			trace = ""
		}
		meta := &requestMeta{route: "unmatched", trace: trace, logger: log}
		switch r.URL.Path {
		case "/healthz", "/readyz", "/auth/login", "/auth/callback", "/auth/session":
			meta.route = "GET " + r.URL.Path
		case "/auth/logout":
			meta.route = "POST /auth/logout"
		}
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
			next.ServeHTTP(apierr.TrackResponse(b), r)
		}()
		if b.status == 0 {
			b.status = 200
		}
		if strings.HasPrefix(r.URL.Path, "/auth/") || strings.HasPrefix(r.URL.Path, "/api/v1/") {
			if b.status == http.StatusOK && meta.route == "GET /api/v1/agent/resolver-documents/{document_id}" {
				b.header.Set("Cache-Control", "private, max-age=0")
			} else {
				b.header.Set("Cache-Control", "no-store")
			}
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
		if requests != nil {
			requests.WithLabelValues(meta.route, strconv.Itoa(b.status)).Inc()
		}
	})
}

// bucketEntry is one token bucket on an intrusive LRU list. head is the most
// recently used entry and tail is the least recently used.
type bucketEntry struct {
	tokens    float64
	at        time.Time
	perSecond float64
	burst     float64
	key       string
	prev      *bucketEntry
	next      *bucketEntry
}

type limiter struct {
	mu      sync.Mutex
	entries map[string]*bucketEntry
	head    *bucketEntry
	tail    *bucketEntry
	max     int
}

func newLimiter(max int) *limiter {
	return &limiter{entries: map[string]*bucketEntry{}, max: max}
}

// allow spends one token or returns how long the caller should wait.
// At capacity it evicts only the least-recently-used bucket, and only when
// that bucket is idle with a full token count or has been idle for more
// than an hour. The check is O(1). A live bucket stays, and the new
// principal is refused. The tail has had the longest time to refill. Ingress
// tables each use one rate. The subject table mixes class rates, so a fuller
// bucket can sit behind a slower tail; that tail stays until it is full or
// an hour old.
func (l *limiter) allow(key string, b config.Bucket) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= l.max && !l.evictIdle(now) {
			return time.Second
		}
		e = &bucketEntry{
			tokens:    float64(b.Burst),
			at:        now,
			perSecond: b.PerSecond,
			burst:     float64(b.Burst),
			key:       key,
		}
		l.entries[key] = e
		l.pushFront(e)
	} else {
		l.moveFront(e)
	}
	e.tokens = math.Min(float64(b.Burst), e.tokens+now.Sub(e.at).Seconds()*b.PerSecond)
	e.at = now
	e.perSecond = b.PerSecond
	e.burst = float64(b.Burst)
	if e.tokens < 1 {
		return time.Duration(math.Ceil((1 - e.tokens) / b.PerSecond * float64(time.Second)))
	}
	e.tokens--
	return 0
}

// evictIdle drops the tail when it has refilled to its burst after going
// idle, or when it has been idle for more than an hour.
func (l *limiter) evictIdle(now time.Time) bool {
	e := l.tail
	if e == nil {
		return false
	}
	elapsed := now.Sub(e.at)
	refilled := math.Min(e.burst, e.tokens+elapsed.Seconds()*e.perSecond)
	fullIdle := e.burst > 0 && refilled >= e.burst && elapsed > 0
	if !fullIdle && elapsed <= time.Hour {
		return false
	}
	l.remove(e)
	delete(l.entries, e.key)
	return true
}

func (l *limiter) pushFront(e *bucketEntry) {
	e.prev = nil
	e.next = l.head
	if l.head != nil {
		l.head.prev = e
	} else {
		l.tail = e
	}
	l.head = e
}

func (l *limiter) remove(e *bucketEntry) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		l.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		l.tail = e.prev
	}
	e.prev = nil
	e.next = nil
}

func (l *limiter) moveFront(e *bucketEntry) {
	if l.head == e {
		return
	}
	l.remove(e)
	l.pushFront(e)
}

// rateClasses are the D-060 route classes. Each class has its own ingress
// table, so an anonymous sweep cannot fill a browser or agent table.
var rateClasses = []string{"browser", "agent", "heartbeat", "lease", "usage", "anonymous"}

type Options struct {
	Config        config.Config
	Sessions      auth.SessionAuthenticator
	Agents        auth.AgentAuthenticator
	AgentListener bool
	Commands      *Commands
	RateLimited   *prometheus.CounterVec
}

// New validates the immutable contract once; security metadata comes from tags,
// security requirements and x-fvpn-auth, including enrollment's explicit exception.
func New(c *Composite, o Options) (http.Handler, error) {
	doc, e := gen.GetSpec()
	if e != nil {
		return nil, e
	}
	doc.Servers = nil
	router, e := legacy.NewRouter(doc)
	if e != nil {
		return nil, e
	}
	if o.Config.MaxRatePrincipals < 1 {
		return nil, errors.New("rate capacity required")
	}
	proxies, e := config.ParseTrustedProxies(o.Config.TrustedProxies)
	if e != nil {
		return nil, e
	}
	subjects := newLimiter(o.Config.MaxRatePrincipals)
	anonymousLim := newLimiter(o.Config.MaxRatePrincipals)
	ingress := make(map[string]*limiter, len(rateClasses))
	for _, class := range rateClasses {
		ingress[class] = newLimiter(o.Config.MaxRatePrincipals)
	}
	var mids []gen.StrictMiddlewareFunc
	if o.Commands != nil {
		mids = append(mids, o.Commands.middleware(c))
	} else {
		mids = append(mids, func(next gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
			return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
				if c.owner(operationID) == nil {
					return nil, errNotImplemented
				}
				return requireCommandBoundary(next, operationID)(ctx, w, r, request)
			}
		})
	}
	strict := gen.NewStrictHandlerWithOptions(c, mids, gen.StrictHTTPServerOptions{RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
		apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Malformed request."))
	}, ResponseErrorHandlerFunc: WriteError})
	handler := gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{BaseURL: "/api/v1", ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
		apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Malformed request."))
	}})
	gates := map[string]chan struct{}{}
	for _, class := range rateClasses {
		gates[class] = make(chan struct{}, o.Config.MaxConcurrent)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = apierr.TrackResponse(w)
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(o.Config.WriteTimeoutSeconds)*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
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
		class, bounds, bodyLimit := classify(op, o.Config)
		// A peer without a verified client certificate never occupies a
		// limiter slot. Per-node buckets are created only after this succeeds.
		var p auth.Principal
		if mtls {
			if o.Agents == nil || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 {
				apierr.Write(w, r, apierr.New(apierr.Unauthenticated, "Verified client certificate required."))
				return
			}
			var authErr error
			p, authErr = o.Agents.AuthenticateAgent(r.Context(), r.TLS.VerifiedChains[0][0])
			if authErr != nil {
				apierr.Write(w, r, authErr)
				return
			}
			if p.Role != auth.Agent || p.NodeID == uuid.Nil || p.IncarnationID == uuid.Nil {
				apierr.Write(w, r, apierr.New(apierr.Unauthenticated, "Invalid machine identity."))
				return
			}
		}
		peer, peerOK := requestRateKey(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), proxies, !o.AgentListener)
		if !peerOK {
			// Do not insert an empty host. Unparseable connection addresses
			// would otherwise share one bucket and could fill the principal map.
			if o.RateLimited != nil {
				o.RateLimited.WithLabelValues(class, "address").Inc()
			}
			apierr.Write(w, r, apierr.New(apierr.RateLimited, "Authentication request rate exceeded.").WithRetryAfter(time.Second))
			return
		}
		// The table is already per class, so the key is the connection principal.
		if delay := ingress[class].allow(peer, o.Config.IngressRate); delay > 0 {
			if o.RateLimited != nil {
				o.RateLimited.WithLabelValues(class, "address").Inc()
			}
			apierr.Write(w, r, apierr.New(apierr.RateLimited, "Authentication request rate exceeded.").WithRetryAfter(delay))
			return
		}
		if !mtls && (tag == "admin" || tag == "member") {
			var authErr error
			if o.Sessions == nil {
				authErr = apierr.New(apierr.Unauthenticated, "Sign in required.")
			} else {
				p, authErr = o.Sessions.AuthenticateSession(r)
			}
			if authErr == nil && (string(p.Role) != tag || p.ID == uuid.Nil) {
				authErr = apierr.New(apierr.Forbidden, "Role is not allowed.")
			}
			if authErr == nil && tag == "admin" && !p.MFAVerified {
				authErr = apierr.New(apierr.Forbidden, "Administrator MFA required.")
			}
			if authErr != nil {
				apierr.Write(w, r, authErr)
				return
			}
		}
		r = r.WithContext(auth.WithPrincipal(r.Context(), p))
		if p.Role == "" {
			// Enrollment and webhooks use this table, separate from every
			// ingress table, so they cannot starve an authenticated class.
			if delay := anonymousLim.allow(peer, bounds); delay > 0 {
				if o.RateLimited != nil {
					o.RateLimited.WithLabelValues(class, "address").Inc()
				}
				apierr.Write(w, r, apierr.New(apierr.RateLimited, "Request rate exceeded.").WithRetryAfter(delay))
				return
			}
		} else if delay := subjects.allow(class+":"+p.Scope(), bounds); delay > 0 {
			if o.RateLimited != nil {
				o.RateLimited.WithLabelValues(class, "subject").Inc()
			}
			apierr.Write(w, r, apierr.New(apierr.RateLimited, "Request rate exceeded.").WithRetryAfter(delay))
			return
		}
		select {
		case gates[class] <- struct{}{}:
			defer func() { <-gates[class] }()
		default:
			if o.RateLimited != nil {
				o.RateLimited.WithLabelValues(class, "concurrency").Inc()
			}
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
		if mutation(r) && (p.Role == auth.Admin || p.Role == auth.Member) || r.Header.Get("Idempotency-Key") != "" {
			values := r.Header.Values("Idempotency-Key")
			_, err := uuid.Parse(r.Header.Get("Idempotency-Key"))
			if err != nil || len(values) != 1 {
				apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "UUID Idempotency-Key is required."))
				return
			}
		}
		for _, par := range op.Parameters {
			if par.Value != nil && par.Value.In == "header" && par.Value.Name == "If-Match" && par.Value.Required && r.Header.Get("If-Match") == "" {
				apierr.Write(w, r, apierr.New(apierr.PreconditionRequired, "If-Match is required."))
				return
			}
		}
		if r.Body == nil {
			r.Body = http.NoBody
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
		if op.OperationID == "ConsumeAgentBootstrap" && len(raw) != 0 {
			apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Bootstrap does not accept a request body."))
			return
		}
		if len(raw) > 0 && tag != "webhook" {
			typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || typ != "application/json" {
				apierr.Write(w, r, apierr.New(apierr.UnsupportedMediaType, "JSON content required."))
				return
			}
			if err = checkJSONDocument(raw, bodyLimit); err != nil {
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
			if err := json.Unmarshal(raw, &body); err != nil {
				apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Invalid JSON document."))
				return
			}
			if !matchingIdentity(body, p) {
				apierr.Write(w, r, apierr.New(apierr.IncarnationMismatch, "Request identity differs from the certificate."))
				return
			}
		}
		// Concrete path parameters are part of the scope. Reusing a key on another
		// resource must never replay the first resource's command (§14.5).
		m.scope = r.Method + " " + route.Path
		for _, name := range parameterNames(route.Path) {
			value := params[name]
			if id, err := uuid.Parse(value); err == nil {
				value = id.String()
			}
			m.scope += "|" + name + "=" + value
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
	route, params, err := router.FindRoute(clone)
	if err == nil {
		return route, params, nil
	}
	// The legacy router allocates RouteError values and cannot classify method
	// mismatches on parameterized paths. Probe only the fixed HTTP method set.
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
		clone.Method = method
		if _, _, e := router.FindRoute(clone); e == nil {
			return nil, nil, routers.ErrMethodNotAllowed
		}
	}
	return nil, nil, err
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

// classify selects a D-060 bucket from the operation id gen.GetSpec returns.
// Codegen stores those ids in PascalCase. The camelCase operationId values in
// contracts/openapi.yaml are not the strings the router loads.
func classify(op *openapi3.Operation, c config.Config) (string, config.Bucket, int64) {
	switch op.OperationID {
	case "SubmitObservation":
		return "heartbeat", c.HeartbeatRate, c.AgentBodyBytes
	case "RenewServingLease":
		return "lease", c.LeaseRate, c.AgentBodyBytes
	case "SubmitUsageBatch":
		return "usage", c.UsageRate, c.UsageBodyBytes
	case "EnrollNode":
		return "anonymous", c.AnonymousRate, c.AgentBodyBytes
	case "ReceiveProviderWebhook":
		return "anonymous", c.AnonymousRate, c.WebhookBodyBytes
	}
	if op.Tags[0] == "agent" {
		return "agent", c.AgentRate, c.AgentBodyBytes
	}
	return "browser", c.BrowserRate, c.BrowserBodyBytes
}

// checkJSONDocument accepts one JSON value. Bodies up to canonjson.MaxBytes
// must be the signed-document subset; that bound stays 1 MiB. A larger body
// is still inside its class cap (usage, 8 MiB) and is checked by
// walkJSONDocument, which applies the same subset rules without a second
// copy of the body. Identity matching reads the parsed body.
func checkJSONDocument(raw []byte, limit int64) error {
	if limit < int64(len(raw)) {
		return errors.New("json: body exceeds class limit")
	}
	if int64(len(raw)) <= canonjson.MaxBytes {
		_, err := canonjson.Canonicalize(raw)
		return err
	}
	return walkJSONDocument(raw)
}

func matchingIdentity(v any, p auth.Principal) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, v := range x {
			if k == "node_id" || k == "node_incarnation_id" || k == "incarnation_id" {
				text, ok := v.(string)
				if !ok {
					return false
				}
				id, e := uuid.Parse(text)
				expected := p.IncarnationID
				if k == "node_id" {
					expected = p.NodeID
				}
				if e != nil || id != expected {
					return false
				}
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

// BrowserRoutes bounds login discovery/token work and session bootstrap. The key
// is the connection principal (IPv4 per address, IPv6 per /64). X-Forwarded-For
// is read only when the TCP peer is inside trusted_proxies.
func BrowserRoutes(c config.Config, next http.Handler) http.Handler {
	return BrowserRoutesWithMetrics(c, next, nil)
}

// ipv6RatePrefixBits is the IPv6 prefix that shares one limiter principal.
// A /64 is one subscriber allocation. Keying the full /128 lets that
// allocation fill MaxRatePrincipals with one fresh bucket per address.
const ipv6RatePrefixBits = 64

// connectionRateKey is the rate-limiter principal for a connection address.
// IPv4, including IPv4-mapped IPv6, is the canonical host. IPv6 is the
// canonical address of its /64. Bracketed "[2001:db8::1]:443" and unbracketed
// "2001:db8::1:443" are the same principal. ok is false when remoteAddr is
// not an IP connection address; the caller fails that request closed and
// does not store a key.
func connectionRateKey(remoteAddr string) (string, bool) {
	return rateKeyFromIP(parseRemoteIP(remoteAddr))
}

func rateKeyFromIP(ip net.IP) (string, bool) {
	if ip == nil {
		return "", false
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String(), true
	}
	prefix := ip.Mask(net.CIDRMask(ipv6RatePrefixBits, 128))
	if prefix == nil {
		return "", false
	}
	return prefix.String(), true
}

// maxForwardedHops is the number of X-Forwarded-For entries read from the
// right. Entries further left are not parsed.
const maxForwardedHops = 64

// requestRateKey is the connection principal for a public-listener request.
// The TCP peer is the principal unless useForwarded is set and that peer
// belongs to a trusted proxy network. Then the principal is the rightmost
// X-Forwarded-For entry that is not itself a trusted hop. A missing header,
// a chain of only trusted hops, or a peer outside the trusted networks keeps
// the TCP peer. An unparseable TCP peer or an unparseable chosen entry
// returns ok false. RFC 7239 Forwarded and X-Real-IP are not read.
// The agent listener passes useForwarded false: ingress stays on the TCP peer.
func requestRateKey(remoteAddr string, forwarded []string, trusted []*net.IPNet, useForwarded bool) (string, bool) {
	if !useForwarded || len(trusted) == 0 {
		return connectionRateKey(remoteAddr)
	}
	peer := parseRemoteIP(remoteAddr)
	if peer == nil {
		return "", false
	}
	if !ipInNets(peer, trusted) {
		return rateKeyFromIP(peer)
	}
	for _, entry := range rightmostForwarded(forwarded, maxForwardedHops) {
		ip := parseForwardedIP(entry)
		if ip == nil {
			return "", false
		}
		if ipInNets(ip, trusted) {
			continue
		}
		return rateKeyFromIP(ip)
	}
	return rateKeyFromIP(peer)
}

func ipInNets(ip net.IP, nets []*net.IPNet) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// rightmostForwarded returns at most limit entries, rightmost first.
// Header values are in on-the-wire order: a later value was appended and is
// further to the right. Only the examined suffix is scanned.
func rightmostForwarded(values []string, limit int) []string {
	if limit < 1 || len(values) == 0 {
		return nil
	}
	out := make([]string, 0, min(limit, 8))
	for i := len(values) - 1; i >= 0 && len(out) < limit; i-- {
		s := values[i]
		end := len(s)
		for len(out) < limit {
			comma := strings.LastIndexByte(s[:end], ',')
			entry := strings.TrimSpace(s[comma+1 : end])
			out = append(out, entry)
			if comma < 0 {
				break
			}
			end = comma
		}
	}
	return out
}

// parseRemoteIP reads the host from a connection address. net.SplitHostPort
// accepts "ipv4:port" and "[ipv6]:port". An unbracketed IPv6 address keeps a
// trailing numeric port ("2001:db8::1:443") so it matches the bracketed form
// of that same host. A zone identifier is not part of the key.
func parseRemoteIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		if h, ok := splitUnbracketedIPv6(remoteAddr); ok {
			host = h
		} else {
			host = remoteAddr
		}
	}
	if i := strings.IndexByte(host, '%'); i >= 0 {
		host = host[:i]
	}
	return net.ParseIP(host)
}

// parseForwardedIP reads one X-Forwarded-For entry. A bracketed address with
// no port ("[2001:db8::1]") is an address. A TCP RemoteAddr in that form stays
// unparseable: connection addresses carry a port.
func parseForwardedIP(entry string) net.IP {
	if len(entry) >= 2 && entry[0] == '[' && entry[len(entry)-1] == ']' {
		entry = entry[1 : len(entry)-1]
	}
	return parseRemoteIP(entry)
}

// splitUnbracketedIPv6 accepts "address:port" when the address contains a
// colon, so net.SplitHostPort rejected it for missing brackets. The port is
// stripped before the prefix is computed. Treating the whole string as one
// IPv6 address would move a trailing port into the host and can change the /64.
func splitUnbracketedIPv6(remoteAddr string) (string, bool) {
	colon := strings.LastIndexByte(remoteAddr, ':')
	if colon <= 0 || colon == len(remoteAddr)-1 {
		return "", false
	}
	host, port := remoteAddr[:colon], remoteAddr[colon+1:]
	if strings.IndexByte(host, ':') < 0 {
		return "", false
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "", false
	}
	bare := host
	if i := strings.IndexByte(bare, '%'); i >= 0 {
		bare = bare[:i]
	}
	if net.ParseIP(bare) == nil {
		return "", false
	}
	return host, true
}

// BrowserRoutesWithMetrics bounds browser authentication and session routes using
// Config.LoginRate (D-060) and records HTTP 429s by endpoint class and limiter.
func BrowserRoutesWithMetrics(c config.Config, next http.Handler, rateLimited *prometheus.CounterVec) http.Handler {
	rate := c.LoginRate
	if rate.PerSecond <= 0 || rate.Burst < 1 {
		rate = config.Defaults().LoginRate
	}
	lim := newLimiter(c.MaxRatePrincipals)
	gate := make(chan struct{}, c.MaxConcurrent)
	proxies, proxyErr := config.ParseTrustedProxies(c.TrustedProxies)
	if proxyErr != nil {
		// Startup validation rejects a bad list. A handler built without it
		// keeps the TCP peer and does not honor a forwarding header.
		proxies = nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = apierr.TrackResponse(w)
		class := "login"
		if r.URL.Path == "/auth/session" || r.URL.Path == "/auth/logout" {
			class = "session"
		}
		host, ok := requestRateKey(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), proxies, true)
		if !ok {
			if rateLimited != nil {
				rateLimited.WithLabelValues(class, "address").Inc()
			}
			apierr.Write(w, r, apierr.New(apierr.RateLimited, "Login request rate exceeded.").WithRetryAfter(time.Second))
			return
		}
		if d := lim.allow(class+":"+host, rate); d > 0 {
			if rateLimited != nil {
				rateLimited.WithLabelValues(class, "address").Inc()
			}
			apierr.Write(w, r, apierr.New(apierr.RateLimited, "Login request rate exceeded.").WithRetryAfter(d))
			return
		}
		select {
		case gate <- struct{}{}:
			defer func() { <-gate }()
		default:
			if rateLimited != nil {
				rateLimited.WithLabelValues(class, "address").Inc()
			}
			apierr.Write(w, r, apierr.New(apierr.RateLimited, "Login concurrency exceeded.").WithRetryAfter(time.Second))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, c.BrowserBodyBytes)
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(c.WriteTimeoutSeconds)*time.Second)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
