//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin"
)

var deviceURLPattern = regexp.MustCompile(`https://netbird\.example\.invalid(?::443)?/[^\s"'<>]+`)

// loginForm is one HTML form from the embedded IdP device flow.
type loginForm struct {
	Action string
	Fields map[string]string
	Links  []string
}

func parseLoginPage(page string) loginForm {
	form := idplogin.Parse(page)
	return loginForm{Action: form.Action, Fields: form.Fields, Links: form.Links}
}

func resolveLoginRef(base *url.URL, ref string) (*url.URL, error) {
	return idplogin.Resolve(base, ref)
}

func deviceFlowURL(log string) string {
	found := deviceURLPattern.FindAllString(log, -1)
	if len(found) == 0 {
		return ""
	}
	return strings.TrimRight(found[len(found)-1], ".,);")
}

func scrubDeviceURL(s string) string {
	return deviceURLPattern.ReplaceAllString(s, "[device-url]")
}

// deviceFlowPath is the URL path only. The query carries the user code.
func deviceFlowPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host != "netbird.example.invalid" {
		return ""
	}
	if u.Path == "" {
		return "/"
	}
	return u.Path
}

// coveredInterval picks a non-empty range inside [from, until]. now is the
// preferred end when it falls inside that window.
func coveredInterval(from, until, now time.Time) (time.Time, time.Time, bool) {
	from = from.UTC()
	until = until.UTC()
	now = now.UTC().Truncate(time.Second)
	if !until.After(from) {
		return time.Time{}, time.Time{}, false
	}
	end := now
	if end.After(until) || !end.After(from) {
		end = until
	}
	start := end.Add(-30 * time.Second)
	if start.Before(from) {
		start = from
	}
	if !end.After(start) || start.Before(from) || end.After(until) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

// completeDeviceLogin walks the embedded IdP screens the way
// deploy/dev/netbird/login-members.py does. It does not record the URL,
// the user code, or the password. The returned status is the first response.
func completeDeviceLogin(ctx context.Context, client *http.Client, start, email, password string) (int, error) {
	return idplogin.Walk(ctx, client, start, email, password, idplogin.DeviceDone)
}
