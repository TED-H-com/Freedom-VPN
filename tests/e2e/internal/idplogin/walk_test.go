package idplogin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndResolveKeepState(t *testing.T) {
	page := `<a href="/oauth2/auth/local?x=1">Local</a>
<form method="post" action="/oauth2/auth/local">
<input type="text" name="login" value="">
<input type="password" name="password" value="">
</form>`
	got := Parse(page)
	if got.Action != "/oauth2/auth/local" || len(got.Links) != 1 {
		t.Fatalf("form: %+v", got)
	}
	base, err := url.Parse("https://netbird.example.invalid/oauth2/auth/local/login?back=%2F&state=page")
	if err != nil {
		t.Fatal(err)
	}
	ref, err := Resolve(base, "/oauth2/auth/local/login?back=%2F&amp;state=kept")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Query().Get("state") != "kept" {
		t.Fatalf("state %q", ref.Query().Get("state"))
	}
}

func TestWalkPostsThePasswordAndLandsOnThePortal(t *testing.T) {
	var posted url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/auth/login":
			http.Redirect(w, r, "/oauth2/auth", http.StatusFound)
		case r.URL.Path == "/oauth2/auth":
			_, _ = w.Write([]byte(`<a href="/oauth2/auth/local?state=abc">Local</a>`))
		case r.URL.Path == "/oauth2/auth/local" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`<form action="/oauth2/auth/local?state=abc&amp;x=1"><input name="login" value=""><input name="password" value=""></form>`))
		case r.URL.Path == "/oauth2/auth/local" && r.Method == http.MethodPost:
			_ = r.ParseForm()
			posted = r.PostForm
			if r.URL.Query().Get("state") != "abc" {
				t.Errorf("state %q", r.URL.Query().Get("state"))
			}
			http.Redirect(w, r, "https://portal.example.invalid:8443/", http.StatusSeeOther)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := srv.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Hostname() == "portal.example.invalid" {
			return http.ErrUseLastResponse
		}
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	}
	status, err := Walk(context.Background(), client, srv.URL+"/auth/login", "user@example.invalid", "s3cret", func(resp *http.Response, _ string) bool {
		return resp.StatusCode == http.StatusSeeOther && resp.Header.Get("Location") == "https://portal.example.invalid:8443/"
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("walk %d %v", status, err)
	}
	if posted.Get("login") != "user@example.invalid" || posted.Get("password") != "s3cret" {
		t.Fatal("password form was not posted")
	}
}

func TestCredForReadsBootstrapFilesAndCreatesTheRest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "netbird"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "netbird", "admin-password"), []byte("admin-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	member := []byte(`{"id":"nb-1","email":"e2e-member@example.invalid","password":"member-secret"}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, "netbird", "member-user.json"), member, 0o600); err != nil {
		t.Fatal(err)
	}
	admin, err := CredFor(context.Background(), nil, "", "", dir, "e2e-admin")
	if err != nil || admin.Email != "admin@example.invalid" || admin.Password != "admin-secret" {
		t.Fatalf("admin: %+v %v", admin, err)
	}
	got, err := CredFor(context.Background(), nil, "", "", dir, "e2e-member")
	if err != nil || got.ID != "nb-1" || got.Email != "e2e-member@example.invalid" {
		t.Fatalf("member: %+v %v", got, err)
	}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token pat" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"id":"nb-new","email":"secops-member-01@example.invalid","password":"generated"}`))
	}))
	defer api.Close()
	created, err := CredFor(context.Background(), api.Client(), api.URL, "pat", dir, "secops-member-01")
	if err != nil || created.ID != "nb-new" || created.Email != "secops-member-01@example.invalid" {
		t.Fatalf("created: %+v %v", created, err)
	}
	again, err := CredFor(context.Background(), nil, "", "", dir, "secops-member-01")
	if err != nil || again.Password != "generated" {
		t.Fatalf("stored: %+v %v", again, err)
	}
	info, err := os.Stat(filepath.Join(dir, "netbird", "users", "secops-member-01.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v %v", info, err)
	}
	if _, err := CredFor(context.Background(), nil, "", "", dir, "missing-user"); err == nil || strings.Contains(err.Error(), "generated") {
		t.Fatalf("missing user error %v", err)
	}
}
