//go:build e2e

package e2e

import (
	"context"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	deviceURLPattern = regexp.MustCompile(`https://netbird\.example\.invalid(?::443)?/[^\s"'<>]+`)
	formTagPattern   = regexp.MustCompile(`(?is)<form\b[^>]*>`)
	inputTagPattern  = regexp.MustCompile(`(?is)<input\b[^>]*>`)
	anchorTagPattern = regexp.MustCompile(`(?is)<a\b[^>]*>`)
	attrPattern      = regexp.MustCompile(`(?i)\b(action|name|value|href)\s*=\s*("([^"]*)"|'([^']*)')`)
)

// loginForm is one HTML form from the embedded IdP device flow.
type loginForm struct {
	Action string
	Fields map[string]string
	Links  []string
}

func parseLoginPage(html string) loginForm {
	var form loginForm
	form.Fields = map[string]string{}
	if m := formTagPattern.FindString(html); m != "" {
		form.Action = attrValue(m, "action")
	}
	for _, tag := range inputTagPattern.FindAllString(html, -1) {
		name := attrValue(tag, "name")
		if name == "" {
			continue
		}
		form.Fields[name] = attrValue(tag, "value")
	}
	for _, tag := range anchorTagPattern.FindAllString(html, -1) {
		if href := attrValue(tag, "href"); href != "" {
			form.Links = append(form.Links, href)
		}
	}
	return form
}

func attrValue(tag, want string) string {
	for _, m := range attrPattern.FindAllStringSubmatch(tag, -1) {
		if !strings.EqualFold(m[1], want) {
			continue
		}
		if m[3] != "" {
			return m[3]
		}
		return m[4]
	}
	return ""
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, start, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	status := resp.StatusCode
	for range 10 {
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			return status, err
		}
		page := string(raw)
		form := parseLoginPage(page)
		if form.Action == "" {
			link := ""
			for _, href := range form.Links {
				if strings.Contains(href, "/auth/local") {
					link = href
					break
				}
			}
			if link != "" {
				next, err := followLogin(ctx, client, resp, link, nil)
				if err != nil {
					return status, err
				}
				resp = next
				continue
			}
			low := strings.ToLower(page)
			if strings.Contains(low, "success") || strings.Contains(low, "logged in") || strings.Contains(low, "device is connected") || strings.Contains(low, "device authorized") {
				return status, nil
			}
			return status, errUnrecognizedLogin
		}
		fields := url.Values{}
		for name, value := range form.Fields {
			fields.Set(name, value)
		}
		for _, name := range []string{"login", "email", "username"} {
			if _, ok := form.Fields[name]; ok {
				fields.Set(name, email)
			}
		}
		if _, ok := form.Fields["password"]; ok {
			fields.Set("password", password)
		}
		if _, ok := form.Fields["approval"]; ok {
			fields.Set("approval", "approve")
		}
		next, err := followLogin(ctx, client, resp, form.Action, strings.NewReader(fields.Encode()))
		if err != nil {
			return status, err
		}
		resp = next
	}
	resp.Body.Close()
	return status, errUnrecognizedLogin
}

var errUnrecognizedLogin = errString("unrecognized embedded IdP screen")

type errString string

func (e errString) Error() string { return string(e) }

// resolveLoginRef resolves a form action or link. html/template escapes
// '&' in the action, and Dex requires the state query on the password POST.
func resolveLoginRef(base *url.URL, ref string) (*url.URL, error) {
	target, err := url.Parse(html.UnescapeString(ref))
	if err != nil {
		return nil, err
	}
	return base.ResolveReference(target), nil
}

func followLogin(ctx context.Context, client *http.Client, prev *http.Response, ref string, body io.Reader) (*http.Response, error) {
	target, err := resolveLoginRef(prev.Request.URL, ref)
	if err != nil {
		return nil, err
	}
	method := http.MethodGet
	if body != nil {
		method = http.MethodPost
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return client.Do(req)
}
