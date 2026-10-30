//go:build e2e && integration

package secops

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin"
)

// §19 AT28/AT31, D-060. Keep only bounded protocol evidence: no callback
// query, redirect Location, credentials, response body or cookie is published.
type loginHTTPRecord struct {
	UTC        time.Time `json:"utc"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	Code       string    `json:"code,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	RetryAfter string    `json:"retry_after,omitempty"`
}

type loginStatusTransport struct {
	base      http.RoundTripper
	last      int
	responses []loginHTTPRecord
}

func (tr *loginStatusTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := tr.base.RoundTrip(r)
	if resp == nil {
		return resp, err
	}
	tr.last = resp.StatusCode
	if r.URL.Host != "portal.example.invalid:8443" || !strings.HasPrefix(r.URL.Path, "/auth/") {
		return resp, err
	}
	record := loginHTTPRecord{UTC: time.Now().UTC(), Method: r.Method, Path: r.URL.Path, Status: resp.StatusCode, RetryAfter: resp.Header.Get("Retry-After"), RequestID: resp.Header.Get("X-Request-ID")}
	if resp.StatusCode == http.StatusTooManyRequests {
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		if readErr == nil {
			var envelope struct {
				Error struct {
					Code      string `json:"code"`
					RequestID string `json:"request_id"`
				} `json:"error"`
			}
			if json.Unmarshal(raw, &envelope) == nil {
				record.Code, record.RequestID = envelope.Error.Code, envelope.Error.RequestID
			}
		}
	}
	tr.responses = append(tr.responses, record)
	return resp, err
}

func (p *liveProbe) loginDefaults() {
	c := config.Defaults()
	raw, err := os.ReadFile(filepath.Join(p.state, "config/central.json"))
	if err != nil || json.Unmarshal(raw, &c) != nil {
		p.t.Fatal("prepared login settings")
	}
	if raw, err = p.docker("control", "printenv", "FVPN_LOGIN_RATE"); err == nil && len(bytes.TrimSpace(raw)) > 0 {
		if json.Unmarshal(raw, &c.LoginRate) != nil {
			p.t.Fatal("login rate override decode")
		}
	}
	if raw, err = p.docker("control", "printenv", "FVPN_LOGIN_SUBJECT_PER_MINUTE"); err == nil && len(bytes.TrimSpace(raw)) > 0 {
		c.LoginSubjectPerMinute, err = strconv.Atoi(strings.TrimSpace(string(raw)))
		if err != nil {
			p.t.Fatal("subject rate override decode")
		}
	}
	p.metadata["login_settings"] = map[string]any{"login_rate": c.LoginRate, "login_subject_per_minute": c.LoginSubjectPerMinute, "max_concurrent": c.MaxConcurrent, "source": "prepared central.json plus named container environment overrides", "mutated": false}
	if c.LoginRate != (config.Bucket{PerSecond: 1, Burst: 40}) || c.LoginSubjectPerMinute != 10 {
		p.t.Fatal("live lab does not use D-060 defaults")
	}
}

func (p *liveProbe) loginMetrics() map[string]int {
	raw, err := p.docker("control", "wget", "-q", "-T", "3", "-O", "-", "http://127.0.0.1:9091/metrics")
	if err != nil {
		p.t.Fatal("control limiter metrics unavailable")
	}
	result := map[string]int{"address": 0, "subject": 0}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "fvpn_control_rate_limited_total{") || !strings.Contains(line, `class="login"`) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			p.t.Fatal("limiter metric shape")
		}
		n, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			p.t.Fatal("limiter metric value")
		}
		for key := range result {
			if strings.Contains(line, `limiter="`+key+`"`) {
				result[key] = int(n)
			}
		}
	}
	return result
}

func (p *liveProbe) rememberCookies(c *http.Client) {
	r, _ := http.NewRequest("GET", livePublic, nil)
	for _, cookie := range c.Jar.Cookies(r.URL) {
		p.secrets = append(p.secrets, []byte(cookie.Value))
	}
}

func (p *liveProbe) loginRefill(seconds int) {
	p.t.Logf("D-060: wait %d seconds for default bucket refill; no setting or clock changes", seconds)
	select {
	case <-p.t.Context().Done():
		p.t.Fatal("login refill cancelled")
	case <-time.After(time.Duration(seconds) * time.Second):
	}
}

func (p *liveProbe) loginThresholds(members []*liveSession) {
	// All requests use the same transport/dial source. Let subject and address
	// buckets refill naturally; do not reset the service or reduce refill rates.
	p.loginRefill(61)
	before := p.loginMetrics()
	member := members[0]
	c := member.client
	trace := &loginStatusTransport{base: p.tr}
	c.Transport = trace
	started := time.Now()
	rejected := 0
	var refusal loginHTTPRecord
	for attempt := 1; attempt <= 15; attempt++ {
		previous := len(trace.responses)
		err := idplogin.PortalLogin(p.t.Context(), c, livePublic, member.cred.Email, member.cred.Password)
		p.rememberCookies(c)
		if len(trace.responses) == previous {
			break
		}
		last := trace.responses[len(trace.responses)-1]
		if err != nil {
			if last.Status == 429 {
				rejected, refusal = attempt, last
			}
			break
		}
	}
	after := p.loginMetrics()
	otherTrace := &loginStatusTransport{base: p.tr}
	other := p.client()
	other.Transport, other.CheckRedirect = otherTrace, idplogin.RedirectLimit(20)
	otherErr := idplogin.PortalLogin(p.t.Context(), other, livePublic, members[1].cred.Email, members[1].cred.Password)
	otherSession := p.call(&liveSession{client: other}, "GET", "/auth/session?role=member", nil, nil)
	var identity struct {
		Authenticated bool   `json:"authenticated"`
		Member        string `json:"member_id"`
	}
	otherOK := otherErr == nil && otherSession.status == 200 && json.Unmarshal(otherSession.body, &identity) == nil && identity.Authenticated && identity.Member == members[1].member
	p.rememberCookies(other)
	subjectPassed := rejected == 11 && refusal.Path == "/auth/callback" && refusal.Code == "RATE_LIMITED" && refusal.RetryAfter != "" && after["subject"]-before["subject"] == 1 && after["address"] == before["address"] && otherOK
	p.metadata["login_subject_limit"] = map[string]any{"expected_first_rejected_completion": 11, "actual_first_rejected_completion": rejected, "first_rejection": refusal, "requests": trace.responses, "elapsed_seconds": time.Since(started).Seconds(), "metrics_before": before, "metrics_after": after, "other_identity_authenticated": otherOK, "other_identity_requests": otherTrace.responses, "passed": subjectPassed}
	if !subjectPassed {
		p.t.Error("D-060 per-identity completion limit or identity independence differs from expected defaults")
	}
	// A warm address bucket permits exactly 40 login initiations inside one
	// refill second. Stop redirects so this measures initiations, not subjects.
	p.loginRefill(41)
	before = p.loginMetrics()
	addressClient := p.client()
	addressTrace := &loginStatusTransport{base: p.tr}
	addressClient.Transport = addressTrace
	addressClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	started = time.Now()
	rejected = 0
	for n := 1; n <= 45; n++ {
		headers := map[string]string{}
		if n > 40 {
			headers["X-Forwarded-For"] = "192.0.2.99"
		}
		r := p.call(&liveSession{client: addressClient}, "GET", "/auth/login", nil, headers)
		p.rememberCookies(addressClient)
		if r.status == 429 {
			rejected = n
			break
		}
		if r.status != 302 {
			p.t.Errorf("login initiation returned %d", r.status)
			break
		}
	}
	addressDuration := time.Since(started)
	after = p.loginMetrics()
	if len(addressTrace.responses) > 0 {
		refusal = addressTrace.responses[len(addressTrace.responses)-1]
	}
	// Session bootstrap is a distinct per-address class even while login is full.
	session := p.call(members[1], "GET", "/auth/session?role=member", nil, nil)
	delay, delayErr := strconv.Atoi(refusal.RetryAfter)
	recovered := false
	if delayErr == nil && delay > 0 && delay <= 60 {
		p.loginRefill(delay)
		r := p.call(&liveSession{client: addressClient}, "GET", "/auth/login", nil, nil)
		recovered = r.status == 302
		p.rememberCookies(addressClient)
	}
	passed := rejected == 41 && addressDuration < time.Second && refusal.Code == "RATE_LIMITED" && delayErr == nil && delay == 1 && after["address"]-before["address"] == 1 && after["subject"] == before["subject"] && session.status == 200 && recovered
	p.metadata["login_41st_request"] = map[string]any{"expected_status": 429, "actual_status": refusal.Status, "expected_first_rejected_request": 41, "actual_first_rejected_request": rejected, "retry_after": refusal.RetryAfter, "first_rejection": refusal, "requests": addressTrace.responses, "elapsed_seconds_before_rejection": addressDuration.Seconds(), "forwarding_header_does_not_bypass": rejected == 41, "session_class_status_while_login_full": session.status, "retry_after_wait_succeeded": recovered, "metrics_before": before, "metrics_after": after, "remeasured": true, "passed": passed}
	if !passed {
		p.t.Error("D-060 address threshold/Retry-After differs from expected default bucket")
	}
}
