package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAssumeTicketsStayOnTheirSession(t *testing.T) {
	s := newTestServer(t)
	srv := httptest.NewTLSServer(s.routes())
	t.Cleanup(srv.Close)
	left := tlsClient(t, srv)
	right := tlsClient(t, srv)
	assume(t, left, srv.URL, "member-a", "member")
	assume(t, right, srv.URL, "member-b", "member")
	if got := exchange(t, left, srv.URL); got != "member-a" {
		t.Fatalf("left subject %q", got)
	}
	if got := exchange(t, right, srv.URL); got != "member-b" {
		t.Fatalf("right subject %q", got)
	}
	if status := authorizeStatus(t, left, srv.URL); status != http.StatusBadRequest {
		t.Fatalf("spent ticket status %d", status)
	}
}

func TestAdapterResignsPortalTokenAndProxiesTheRest(t *testing.T) {
	dexKey := mustKey(t)
	adapter := newTestServer(t)
	const (
		issuer = "https://netbird.example.invalid/oauth2"
		sub    = "dex-subject"
		nonce  = "nonce-1"
	)
	var deviceBody string
	dex := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/keys":
			writeJSON(w, http.StatusOK, map[string]any{"keys": []any{jwkAny(publicJWK(dexKey, "dex-1"))}})
		case "/oauth2/token":
			raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			values, _ := url.ParseQuery(string(raw))
			if values.Get("grant_type") == "urn:ietf:params:oauth:grant-type:device_code" {
				deviceBody = string(raw)
				if r.Header.Get("Authorization") != "Bearer device" {
					t.Errorf("device authorization header = %q", r.Header.Get("Authorization"))
				}
				writeJSON(w, http.StatusOK, map[string]any{"access_token": "device-access", "token_type": "Bearer"})
				return
			}
			if r.Header.Get("Authorization") != "" || values.Get("client_secret") != "" {
				t.Errorf("portal secret reached dex")
			}
			if values.Get("redirect_uri") != adapter.redirect {
				writeJSON(w, http.StatusOK, map[string]any{"id_token": "left-as-dex", "access_token": "dash"})
				return
			}
			tok, err := signRS256(dexKey, "dex-1", map[string]any{
				"iss": issuer, "sub": sub, "aud": adapter.portalClient, "nonce": nonce,
				"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
				"name": "E2E Admin", "at_hash": "hash-1",
			})
			if err != nil {
				t.Errorf("sign dex: %v", err)
				http.Error(w, "bad", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"id_token": tok, "access_token": "dex-access", "token_type": "Bearer"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(dex.Close)
	adapter.dexUpstream = dex.URL
	adapter.embedIssuer = issuer
	adapter.portalClient = "netbird-dashboard"
	adapter.dexHost = ""
	adapter.dexClient = dex.Client()
	front := httptest.NewServer(adapter.routes())
	t.Cleanup(front.Close)

	portal := postForm(t, http.DefaultClient, front.URL+"/oauth2/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {"code-1"}, "redirect_uri": {adapter.redirect},
		"code_verifier": {"verifier"},
	}, "netbird-dashboard", "not-sent-to-dex")
	if portal.StatusCode != http.StatusOK {
		t.Fatalf("portal status %d", portal.StatusCode)
	}
	var body map[string]any
	if json.Unmarshal(portal.Body, &body) != nil || body["access_token"] != "dex-access" {
		t.Fatalf("portal body %s", portal.Body)
	}
	idToken, _ := body["id_token"].(string)
	claims, err := verifyRS256(idToken, func(kid string) (*rsa.PublicKey, error) {
		if kid != "e2e-portal" {
			t.Fatalf("kid %s", kid)
		}
		return &adapter.key.PublicKey, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if claimString(claims["sub"]) != sub || claimString(claims["iss"]) != issuer || claimString(claims["nonce"]) != nonce || claimString(claims["at_hash"]) != "hash-1" {
		t.Fatalf("claims %s", portal.Body)
	}
	if claimString(claims["aud"]) != "netbird-dashboard" {
		t.Fatalf("aud %s", claims["aud"])
	}
	var roles []string
	if json.Unmarshal(claims["roles"], &roles) != nil || len(roles) != 1 || roles[0] != "admin" {
		t.Fatalf("roles %s", claims["roles"])
	}
	var amr []string
	if json.Unmarshal(claims["amr"], &amr) != nil || len(amr) != 1 || amr[0] != "mfa" {
		t.Fatalf("amr %s", claims["amr"])
	}
	if claimInt(claims["auth_time"]) == 0 || claimInt(claims["auth_time"]) != claimInt(claims["iat"]) {
		t.Fatal("auth_time was not copied from iat")
	}
	if _, err := verifyRS256(idToken, func(string) (*rsa.PublicKey, error) { return &dexKey.PublicKey, nil }); err == nil {
		t.Fatal("dex key verified the portal token")
	}

	other := postForm(t, http.DefaultClient, front.URL+"/oauth2/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {"code-2"}, "redirect_uri": {"https://netbird.example.invalid/nb-auth"},
		"client_id": {"netbird-dashboard"},
	}, "", "")
	if !strings.Contains(string(other.Body), "left-as-dex") {
		t.Fatalf("dashboard token was rewritten: %s", other.Body)
	}
	deviceRaw := "grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code&device_code=abc&client_id=netbird-cli"
	req, err := http.NewRequest(http.MethodPost, front.URL+"/oauth2/token", strings.NewReader(deviceRaw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer device")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(got), "device-access") || deviceBody != deviceRaw {
		t.Fatalf("device proxy status %d body %s upstream %q", resp.StatusCode, got, deviceBody)
	}
	keysResp, err := http.Get(front.URL + "/oauth2/keys")
	if err != nil {
		t.Fatal(err)
	}
	defer keysResp.Body.Close()
	keysRaw, _ := io.ReadAll(keysResp.Body)
	if !strings.Contains(string(keysRaw), "dex-1") || !strings.Contains(string(keysRaw), "e2e-portal") {
		t.Fatalf("jwks %s", keysRaw)
	}
}

func TestMemberProfileStaysAMember(t *testing.T) {
	if got := (&server{adminEmail: "admin@example.invalid", adminName: "E2E Admin"}).isAdmin("e2e-member", "e2e-member@example.invalid", "e2e-member"); got {
		t.Fatal("member classified as admin")
	}
	if !(&server{adminEmail: "admin@example.invalid", adminName: "E2E Admin"}).isAdmin("admin@example.invalid", "", "") {
		t.Fatal("admin email in the name claim was not recognized")
	}
	if (&server{adminEmail: "admin@example.invalid", adminName: "E2E Admin"}).isAdmin("", "", "") {
		t.Fatal("empty profile was classified as admin")
	}
}

func newTestServer(t *testing.T) *server {
	t.Helper()
	return &server{
		issuer: "https://idp.example.invalid", clientID: "portal-e2e", clientSecret: "secret",
		redirect: "https://portal.example.invalid:8443/auth/callback", assume: "assume-secret",
		key: mustKey(t), kid: "e2e-1", portalKid: "e2e-portal",
		adminEmail: "admin@example.invalid", adminName: "E2E Admin",
		dexClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func mustKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func publicJWK(key *rsa.PrivateKey, kid string) map[string]string {
	s := &server{key: key}
	return s.publicJWK(kid)
}

func tlsClient(t *testing.T, srv *httptest.Server) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	base := srv.Client()
	return &http.Client{
		Transport:     base.Transport,
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func assume(t *testing.T, client *http.Client, base, subject, role string) {
	t.Helper()
	body := `{"subject":"` + subject + `","role":"` + role + `","name":"` + subject + `"}`
	req, err := http.NewRequest(http.MethodPost, base+"/e2e/assume", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-E2E-Secret", "assume-secret")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("X-E2E-Ticket") == "" {
		t.Fatalf("assume %d", resp.StatusCode)
	}
}

func exchange(t *testing.T, client *http.Client, base string) string {
	t.Helper()
	verifier := "verifier-value"
	q := url.Values{
		"client_id": {clientIDOf(t)}, "redirect_uri": {"https://portal.example.invalid:8443/auth/callback"},
		"response_type": {"code"}, "code_challenge_method": {"S256"}, "code_challenge": {sha256Sum(verifier)},
		"state": {"state-1"}, "nonce": {"nonce-1"},
	}
	resp, err := client.Get(base + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("authorize %d", resp.StatusCode)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil || loc.Query().Get("code") == "" {
		t.Fatalf("location %q", resp.Header.Get("Location"))
	}
	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {loc.Query().Get("code")},
		"redirect_uri": {"https://portal.example.invalid:8443/auth/callback"}, "code_verifier": {verifier},
		"client_id": {"portal-e2e"}, "client_secret": {"secret"},
	}
	tok := postForm(t, client, base+"/token", form, "", "")
	if tok.StatusCode != http.StatusOK {
		t.Fatalf("token %d %s", tok.StatusCode, tok.Body)
	}
	var payload struct {
		IDToken string `json:"id_token"`
	}
	if json.Unmarshal(tok.Body, &payload) != nil {
		t.Fatal("token json")
	}
	parts := strings.Split(payload.IDToken, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims struct {
		Subject string `json:"sub"`
	}
	if json.Unmarshal(raw, &claims) != nil || claims.Subject == "" {
		t.Fatalf("claims %s", raw)
	}
	return claims.Subject
}

func clientIDOf(*testing.T) string { return "portal-e2e" }

func authorizeStatus(t *testing.T, client *http.Client, base string) int {
	t.Helper()
	q := url.Values{
		"client_id": {"portal-e2e"}, "redirect_uri": {"https://portal.example.invalid:8443/auth/callback"},
		"response_type": {"code"}, "code_challenge_method": {"S256"}, "code_challenge": {sha256Sum("verifier-value")},
		"state": {"state-1"}, "nonce": {"nonce-1"},
	}
	resp, err := client.Get(base + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

type recordedBody struct {
	StatusCode int
	Body       []byte
}

func postForm(t *testing.T, client *http.Client, rawURL string, form url.Values, user, password string) recordedBody {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if user != "" {
		req.SetBasicAuth(user, password)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		t.Fatal(err)
	}
	return recordedBody{StatusCode: resp.StatusCode, Body: body}
}
