// Package idplogin walks the lab embedded IdP the way a browser does.
// Passwords stay in the POST body and are not returned in errors.
package idplogin

import (
	"context"
	"errors"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	formTagPattern   = regexp.MustCompile(`(?is)<form\b[^>]*>`)
	inputTagPattern  = regexp.MustCompile(`(?is)<input\b[^>]*>`)
	anchorTagPattern = regexp.MustCompile(`(?is)<a\b[^>]*>`)
	attrPattern      = regexp.MustCompile(`(?i)\b(action|name|value|href)\s*=\s*("([^"]*)"|'([^']*)')`)
)

// ErrUnrecognized is returned when the page is neither a login form nor a
// completed sign-in.
var ErrUnrecognized = errors.New("unrecognized embedded IdP screen")

// Form is one HTML form from the embedded IdP.
type Form struct {
	Action string
	Fields map[string]string
	Links  []string
}

// Parse reads the first form, its inputs, and the anchors on the page.
func Parse(page string) Form {
	var form Form
	form.Fields = map[string]string{}
	if m := formTagPattern.FindString(page); m != "" {
		form.Action = attrValue(m, "action")
	}
	for _, tag := range inputTagPattern.FindAllString(page, -1) {
		name := attrValue(tag, "name")
		if name == "" {
			continue
		}
		form.Fields[name] = attrValue(tag, "value")
	}
	for _, tag := range anchorTagPattern.FindAllString(page, -1) {
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

// Resolve resolves a form action or link. html/template escapes '&' in the
// action, and Dex requires the state query on the password POST.
func Resolve(base *url.URL, ref string) (*url.URL, error) {
	target, err := url.Parse(html.UnescapeString(ref))
	if err != nil {
		return nil, err
	}
	return base.ResolveReference(target), nil
}

// Walk follows login pages until accept reports success. The returned status
// is the first response. accept sees each page that has no form to submit.
func Walk(ctx context.Context, client *http.Client, start, email, password string, accept func(*http.Response, string) bool) (int, error) {
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
		// A completed callback lands on the served portal with HTTP 200.
		// Do not submit an unrelated form from that landing page.
		if portalHost(resp) && accept != nil && accept(resp, page) {
			return status, nil
		}
		form := Parse(page)
		if form.Action == "" {
			link := ""
			for _, href := range form.Links {
				if strings.Contains(href, "/auth/local") {
					link = href
					break
				}
			}
			if link != "" {
				next, err := follow(ctx, client, resp, link, nil)
				if err != nil {
					return status, err
				}
				resp = next
				continue
			}
			if accept != nil && accept(resp, page) {
				return status, nil
			}
			return status, ErrUnrecognized
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
		next, err := follow(ctx, client, resp, form.Action, strings.NewReader(fields.Encode()))
		if err != nil {
			return status, err
		}
		resp = next
	}
	resp.Body.Close()
	return status, ErrUnrecognized
}

func follow(ctx context.Context, client *http.Client, prev *http.Response, ref string, body io.Reader) (*http.Response, error) {
	if prev.Request == nil || prev.Request.URL == nil {
		return nil, errors.New("login response has no url")
	}
	target, err := Resolve(prev.Request.URL, ref)
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

// DeviceDone reports a completed device-flow page.
func DeviceDone(_ *http.Response, page string) bool {
	low := strings.ToLower(page)
	return strings.Contains(low, "success") || strings.Contains(low, "logged in") || strings.Contains(low, "device is connected") || strings.Contains(low, "device authorized")
}

func portalHost(resp *http.Response) bool {
	return resp != nil && resp.Request != nil && resp.Request.URL != nil && resp.Request.URL.Hostname() == "portal.example.invalid"
}

// PortalDone requires a healthy landing after the callback. Callback errors,
// redirects and a missing or failing index are not successful logins.
func PortalDone(resp *http.Response, _ string) bool {
	return portalHost(resp) && resp.StatusCode == http.StatusOK &&
		resp.Request.URL.Path != "/auth/callback" &&
		!strings.HasPrefix(resp.Request.URL.Path, "/auth/")
}

// PortalLogin starts at the portal login route and finishes on the portal.
func PortalLogin(ctx context.Context, client *http.Client, portalBase, email, password string) error {
	_, err := Walk(ctx, client, strings.TrimRight(portalBase, "/")+"/auth/login", email, password, PortalDone)
	return err
}

// RedirectLimit raises the client's redirect cap for the embedded login hops.
func RedirectLimit(max int) func(*http.Request, []*http.Request) error {
	return func(_ *http.Request, via []*http.Request) error {
		if len(via) >= max {
			return errors.New("stopped after too many redirects")
		}
		return nil
	}
}
