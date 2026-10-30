// Package httpapi is the contract-driven transport and transaction boundary.
package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
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
	clientCert   *x509.Certificate
	clientDomain string
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

// ClientCertificate returns the authenticated mTLS leaf certificate for an
// agent-listener request, or nil on other listeners.
func ClientCertificate(ctx context.Context) *x509.Certificate {
	if m, ok := ctx.Value(metaKey{}).(*requestMeta); ok {
		return m.clientCert
	}
	return nil
}

// ClientCompatibilityDomain returns the coarsened client endpoint domain
// (IPv4 /24 or IPv6 /48, honouring X-Forwarded-For only from configured
// trusted proxies) for an authenticated mTLS request, or "" when unavailable.
func ClientCompatibilityDomain(ctx context.Context) string {
	if m, ok := ctx.Value(metaKey{}).(*requestMeta); ok {
		return m.clientDomain
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
	group     string
	prev      *bucketEntry
	next      *bucketEntry
}

type limiter struct {
	mu      sync.Mutex
	entries map[string]*bucketEntry
	groups  map[string]int
	head    *bucketEntry
	tail    *bucketEntry
	max     int
}

func newLimiter(max int) *limiter {
	return &limiter{entries: map[string]*bucketEntry{}, groups: map[string]int{}, max: max}
}

// limiterNow is the clock for token refill and idle eviction. Production
// leaves it as time.Now. A test replaces it so eligibility for eviction does
// not depend on how fast the requests run, and restores it when the test ends.
var limiterNow = time.Now

// evictIdleWindow is how many least-recently-used buckets are examined when
// the table is full. A constant window stays O(1). One depleted tail cannot
// hide a newer bucket that has already refilled. More than this many depleted
// buckets sitting at the tail can still refuse a newcomer while a full bucket
// waits further toward the head.
const evictIdleWindow = 8

// allow spends one token or returns how long the caller should wait.
// At capacity it scans a bounded window from the tail and evicts the first
// bucket that is idle with a full token count or has been idle for more than
// an hour. A live bucket stays. When no bucket in the window qualifies, the
// new principal is refused and nothing is inserted.
func (l *limiter) allow(key string, b config.Bucket) time.Duration {
	return l.allowGrouped(key, "", b)
}

// allowGrouped is allow with an optional group budget. An empty group is not
// counted. A new key whose group is already at the budget is refused without
// eviction and without an insert. An existing key still spends a token.
func (l *limiter) allowGrouped(key, group string, b config.Bucket) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := limiterNow()
	e, ok := l.entries[key]
	if !ok {
		if group != "" && l.groups[group] >= l.groupBudget() {
			return time.Second
		}
		if len(l.entries) >= l.max && !l.evictIdle(now) {
			return time.Second
		}
		e = &bucketEntry{
			tokens:    float64(b.Burst),
			at:        now,
			perSecond: b.PerSecond,
			burst:     float64(b.Burst),
			key:       key,
			group:     group,
		}
		l.entries[key] = e
		if group != "" {
			l.groups[group]++
		}
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

// groupBudget is the most /64 keys one anonymous /48 may hold. Half the
// principal cap leaves room for another prefix. A cap of one still allows
// that single slot.
func (l *limiter) groupBudget() int {
	n := l.max / 2
	if n < 1 {
		return 1
	}
	return n
}

// evictIdle drops the first eligible bucket in a window that starts at the
// tail and walks toward newer entries. Eligible means the bucket has refilled
// to its burst after going idle, or it has been idle for more than an hour.
func (l *limiter) evictIdle(now time.Time) bool {
	e := l.tail
	for n := 0; e != nil && n < evictIdleWindow; n++ {
		elapsed := now.Sub(e.at)
		refilled := math.Min(e.burst, e.tokens+elapsed.Seconds()*e.perSecond)
		fullIdle := e.burst > 0 && refilled >= e.burst && elapsed > 0
		if fullIdle || elapsed > time.Hour {
			l.drop(e)
			return true
		}
		e = e.prev
	}
	return false
}

func (l *limiter) drop(e *bucketEntry) {
	l.remove(e)
	delete(l.entries, e.key)
	if e.group == "" {
		return
	}
	l.groups[e.group]--
	if l.groups[e.group] <= 0 {
		delete(l.groups, e.group)
	}
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
		// limiter slot. The agent listener then spends an ingress token and a
		// concurrency slot, both keyed by that certificate, before the
		// database lookup. A failed lookup still spent the token.
		var p auth.Principal
		var peer string
		if mtls {
			if o.Agents == nil || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 {
				apierr.Write(w, r, apierr.New(apierr.Unauthenticated, "Verified client certificate required."))
				return
			}
			leaf := r.TLS.VerifiedChains[0][0]
			if delay := ingress[class].allow(certRateKey(leaf), o.Config.IngressRate); delay > 0 {
				if o.RateLimited != nil {
					o.RateLimited.WithLabelValues(class, "address").Inc()
				}
				apierr.Write(w, r, apierr.New(apierr.RateLimited, "Authentication request rate exceeded.").WithRetryAfter(delay))
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
			var authErr error
			p, authErr = o.Agents.AuthenticateAgent(r.Context(), leaf)
			if authErr != nil {
				apierr.Write(w, r, authErr)
				return
			}
			if p.Role != auth.Agent || p.NodeID == uuid.Nil || p.IncarnationID == uuid.Nil {
				apierr.Write(w, r, apierr.New(apierr.Unauthenticated, "Invalid machine identity."))
				return
			}
			m.clientCert = leaf
			if domain, ok := compatibilityDomain(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), proxies); ok {
				m.clientDomain = domain
			}
		} else {
			var peerOK bool
			peer, peerOK = requestRateKey(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), proxies, true)
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
			// Anonymous enrollment and webhooks also count the IPv6 /48.
			group := ""
			if class == "anonymous" {
				group = anonymousGroup(peer)
			}
			if delay := ingress[class].allowGrouped(peer, group, o.Config.IngressRate); delay > 0 {
				if o.RateLimited != nil {
					o.RateLimited.WithLabelValues(class, "address").Inc()
				}
				apierr.Write(w, r, apierr.New(apierr.RateLimited, "Authentication request rate exceeded.").WithRetryAfter(delay))
				return
			}
			if tag == "admin" || tag == "member" {
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
		}
		r = r.WithContext(auth.WithPrincipal(r.Context(), p))
		if p.Role == "" {
			// Enrollment and webhooks use this table, separate from every
			// ingress table, so they cannot starve an authenticated class.
			// The /48 budget sits beside the /64 key on this table too.
			group := ""
			if class == "anonymous" {
				group = anonymousGroup(peer)
			}
			if delay := anonymousLim.allowGrouped(peer, group, bounds); delay > 0 {
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
		if !mtls {
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
		identOK := true
		streamedSchema := false
		if len(raw) > 0 && tag != "webhook" {
			typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || typ != "application/json" {
				apierr.Write(w, r, apierr.New(apierr.UnsupportedMediaType, "JSON content required."))
				return
			}
			chk := &jsonCheck{}
			if mtls {
				chk.onIdentity = func(key, value string, isString bool) {
					if !identityFieldMatches(key, value, isString, p) {
						identOK = false
					}
				}
			}
			if int64(len(raw)) <= canonjson.MaxBytes {
				if _, err = canonjson.Canonicalize(raw); err != nil {
					apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Invalid JSON document."))
					return
				}
				if chk.onIdentity != nil {
					if err = walkChecked(raw, chk); err != nil {
						apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Invalid JSON document."))
						return
					}
				}
			} else if err = checkLargeJSON(raw, bodyLimit, op, chk, &streamedSchema); err != nil {
				if errors.Is(err, errSchemaMismatch) {
					apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Request does not match the API schema."))
					return
				}
				apierr.Write(w, r, apierr.New(apierr.MalformedRequest, "Invalid JSON document."))
				return
			}
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		validation := &openapi3filter.RequestValidationInput{Request: r, PathParams: params, Route: route, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc, SkipSettingDefaults: true, ExcludeRequestBody: tag == "webhook" || streamedSchema, RejectWhenRequestBodyNotSpecified: true}}
		if e = openapi3filter.ValidateRequest(r.Context(), validation); e != nil {
			apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Request does not match the API schema."))
			return
		}
		if mtls && len(raw) > 0 && !identOK {
			apierr.Write(w, r, apierr.New(apierr.IncarnationMismatch, "Request identity differs from the certificate."))
			return
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
		// An unimplemented operation has already passed the streaming checks.
		// The generated handler would decode the body into a struct and then
		// report that it has no owner. Skip that decode.
		if o.Commands == nil && c.owner(op.OperationID) == nil {
			WriteError(w, r, errNotImplemented)
			return
		}
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

// checkLargeJSON walks a body above canonjson.MaxBytes. When the operation
// schema is covered by the streaming checker, that walk is the schema check
// and streamed is set so the caller does not build a second document tree.
func checkLargeJSON(raw []byte, limit int64, op *openapi3.Operation, chk *jsonCheck, streamed *bool) error {
	if limit < int64(len(raw)) {
		return errors.New("json: body exceeds class limit")
	}
	if schema := requestJSONSchema(op); schema != nil && schemaStreamable(schema) {
		chk.schema = schema
		*streamed = true
	}
	return walkChecked(raw, chk)
}

func requestJSONSchema(op *openapi3.Operation) *openapi3.Schema {
	if op == nil || op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil
	}
	media := op.RequestBody.Value.Content.Get("application/json")
	if media == nil || media.Schema == nil {
		return nil
	}
	return media.Schema.Value
}

func identityFieldMatches(key, value string, isString bool, p auth.Principal) bool {
	if key != "node_id" && key != "node_incarnation_id" && key != "incarnation_id" {
		return true
	}
	if !isString {
		return false
	}
	id, err := uuid.Parse(value)
	expected := p.IncarnationID
	if key == "node_id" {
		expected = p.NodeID
	}
	return err == nil && id == expected
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
// canonical address of its /64. An unbracketed string that parses as an IPv6
// address is that address; a trailing port is removed only when the full
// string does not parse. ok is false when remoteAddr is not an IP connection
// address; the caller fails that request closed and does not store a key.
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

// ipv6AggregatePrefixBits is the anonymous-table budget beside the /64 key.
// One /48 may hold at most half of max_rate_principals /64 buckets.
const ipv6AggregatePrefixBits = 48

// anonymousGroup is the /48 of an IPv6 rate key. IPv4 keys are not grouped.
func anonymousGroup(rateKey string) string {
	ip := net.ParseIP(rateKey)
	if ip == nil || ip.To4() != nil {
		return ""
	}
	masked := ip.Mask(net.CIDRMask(ipv6AggregatePrefixBits, 128))
	if masked == nil {
		return ""
	}
	return masked.String()
}

// certRateKey is the SHA-256 of the verified leaf certificate's DER. It is
// computed from memory and is the agent listener's ingress key.
func certRateKey(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
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
// The agent listener's rate-limiter ingress key is the SHA-256 of the verified
// client certificate; it calls compatibilityDomain (which wraps requestRateKey)
// to derive the coarsened /24 or /48 client domain for cloned-identity checks (D-072).
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

const ipv4CompatibilityPrefixBits = 24

// compatibilityDomain derives the endpoint compatibility domain (D-072) for an
// authenticated request using the trusted-proxy logic of requestRateKey and
// coarsening IPv4 to /24 and IPv6 to /48 before server-side hashing. Two clones
// behind the same compatibility domain are not distinguished by this check.
func compatibilityDomain(remoteAddr string, forwarded []string, trusted []*net.IPNet) (string, bool) {
	key, ok := requestRateKey(remoteAddr, forwarded, trusted, true)
	if !ok {
		return "", false
	}
	ip := net.ParseIP(key)
	if ip == nil {
		return "", false
	}
	if v4 := ip.To4(); v4 != nil {
		masked := v4.Mask(net.CIDRMask(ipv4CompatibilityPrefixBits, 32))
		if masked == nil {
			return "", false
		}
		return masked.String() + "/24", true
	}
	masked := ip.Mask(net.CIDRMask(ipv6AggregatePrefixBits, 128))
	if masked == nil {
		return "", false
	}
	return masked.String() + "/48", true
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
// accepts "ipv4:port" and "[ipv6]:port". A fully bracketed address with no
// port is not a connection address. An unbracketed value with more than one
// colon is parsed as an IPv6 address first; a trailing :port is removed only
// when that parse fails. A zone identifier is not part of the key.
func parseRemoteIP(remoteAddr string) net.IP {
	return parseIPText(remoteAddr, false)
}

// parseForwardedIP reads one X-Forwarded-For entry. A bracketed address with
// no port ("[2001:db8::1]") is an address and is never split. A TCP RemoteAddr
// in that form stays unparseable: connection addresses carry a port.
func parseForwardedIP(entry string) net.IP {
	return parseIPText(entry, true)
}

func parseIPText(s string, bareBracketOK bool) net.IP {
	if len(s) >= 2 && s[0] == '[' {
		if s[len(s)-1] == ']' {
			if !bareBracketOK {
				return nil
			}
			return parseIPHost(s[1 : len(s)-1])
		}
		host, _, err := net.SplitHostPort(s)
		if err != nil {
			return nil
		}
		return parseIPHost(host)
	}
	if strings.Count(s, ":") > 1 {
		if ip := parseIPHost(s); ip != nil {
			return ip
		}
		host, port, ok := splitTrailingPort(s)
		if !ok {
			return nil
		}
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return nil
		}
		return parseIPHost(host)
	}
	if host, _, err := net.SplitHostPort(s); err == nil {
		return parseIPHost(host)
	}
	return parseIPHost(s)
}

func parseIPHost(host string) net.IP {
	if i := strings.IndexByte(host, '%'); i >= 0 {
		host = host[:i]
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip
	}
	// Eight hextets written with a zero-width "::" (2001:db8:1::2:3:4:5:6
	// for 2001:db8:1:2:3:4:5:6) are that address. net.ParseIP rejects the
	// zero-width marker. Accepting it keeps the last hextet in the host, so
	// it cannot be read as a port and move the /64.
	return parseZeroWidthCompression(host)
}

func parseZeroWidthCompression(host string) net.IP {
	if strings.Count(host, "::") != 1 {
		return nil
	}
	n := 0
	for _, p := range strings.Split(host, ":") {
		if p != "" {
			n++
		}
	}
	if n != 8 {
		return nil
	}
	return net.ParseIP(strings.Replace(host, "::", ":", 1))
}

// splitTrailingPort separates a final :port from an unbracketed IPv6 literal
// after the whole string failed to parse. The host is not required to parse
// here; the caller checks it.
func splitTrailingPort(s string) (host, port string, ok bool) {
	colon := strings.LastIndexByte(s, ':')
	if colon <= 0 || colon == len(s)-1 {
		return "", "", false
	}
	host, port = s[:colon], s[colon+1:]
	if strings.IndexByte(host, ':') < 0 {
		return "", "", false
	}
	return host, port, true
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
