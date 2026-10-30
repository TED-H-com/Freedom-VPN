// Command fvpn-e2e-nbbootstrap performs the TEST ONLY NetBird setup that
// deploy/dev/netbird/bootstrap.py does, against this stack's HTTPS endpoint.
// It writes tokens into the state directory and prints no secret.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	api := flag.String("api", "https://netbird.example.invalid/api", "management API base")
	dial := flag.String("dial", "", "host:port to dial instead of DNS")
	caFile := flag.String("ca", "", "lab CA bundle")
	state := flag.String("state", "", "state directory")
	flag.Parse()
	if *caFile == "" || *state == "" {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-nbbootstrap: -ca and -state are required")
		os.Exit(2)
	}
	client, err := httpClient(*api, *dial, *caFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-nbbootstrap: client")
		os.Exit(1)
	}
	if err := bootstrap(client, strings.TrimRight(*api, "/"), *state); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-nbbootstrap:", err.Error())
		os.Exit(1)
	}
	fmt.Println("netbird bootstrap verified")
}

func httpClient(api, dialAddr, caFile string) (*http.Client, error) {
	pemBytes, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("ca")
	}
	parsed, err := url.Parse(api)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12, ServerName: parsed.Hostname()}}
	if dialAddr != "" {
		tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, dialAddr)
		}
	}
	return &http.Client{Transport: tr, Timeout: 20 * time.Second}, nil
}

func call(client *http.Client, method, url, token string, body any) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Token "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("netbird unavailable: %s", err.Error())
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("netbird unavailable: %s", err.Error())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(httpFailure(resp.StatusCode, raw, ""))
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	return raw, nil
}

// httpFailure is the status and redacted body of a call that did not yield a token.
// Secret fields are replaced so a 2xx setup response cannot print a PAT.
func httpFailure(status int, raw []byte, note string) string {
	text := redactBody(raw)
	if text == "" {
		text = "empty body"
	}
	msg := fmt.Sprintf("netbird HTTP %d: %s", status, text)
	if note != "" {
		msg += " (" + note + ")"
	}
	return msg
}

func redactBody(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	var doc any
	if json.Unmarshal(raw, &doc) == nil {
		redactJSON(doc)
		if out, err := json.Marshal(doc); err == nil {
			text = string(out)
		}
	}
	if len(text) > 500 {
		text = text[:500]
	}
	return text
}

func redactJSON(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			switch strings.ToLower(k) {
			case "password", "personal_access_token", "plain_token", "token", "secret", "setup_key":
				if _, ok := val.(string); ok && val != "" {
					t[k] = "[redacted]"
				}
			default:
				redactJSON(val)
			}
		}
	case []any:
		for _, item := range t {
			redactJSON(item)
		}
	}
}

func bootstrap(client *http.Client, api, state string) error {
	nb := filepath.Join(state, "netbird")
	if err := os.MkdirAll(nb, 0o700); err != nil {
		return err
	}
	var instance struct {
		SetupRequired bool `json:"setup_required"`
	}
	var last error
	for attempt := 0; attempt < 60; attempt++ {
		raw, err := call(client, http.MethodGet, api+"/instance", "", nil)
		if err == nil {
			if json.Unmarshal(raw, &instance) != nil {
				return fmt.Errorf("netbird instance")
			}
			last = nil
			break
		}
		last = err
		time.Sleep(time.Second)
	}
	if last != nil {
		return last
	}
	ownerPath := filepath.Join(nb, "owner-token")
	if instance.SetupRequired {
		password, err := randomToken()
		if err != nil {
			return err
		}
		if err := writeSecret(filepath.Join(nb, "admin-password"), password); err != nil {
			return err
		}
		raw, err := call(client, http.MethodPost, api+"/setup", "", map[string]any{
			"email": "admin@example.invalid", "name": "E2E Admin", "password": password,
			"create_pat": true, "pat_expire_in": 1,
		})
		if err != nil {
			return err
		}
		var setup struct {
			PersonalAccessToken string `json:"personal_access_token"`
		}
		if json.Unmarshal(raw, &setup) != nil || setup.PersonalAccessToken == "" {
			// Embedded setup, not the lab IdP. personal_access_token is omitted
			// unless the server was started with NB_SETUP_PAT_ENABLED=true.
			return errors.New(httpFailure(http.StatusOK, raw, "personal_access_token missing"))
		}
		if err := writeSecret(ownerPath, setup.PersonalAccessToken); err != nil {
			return err
		}
	}
	owner, err := os.ReadFile(ownerPath)
	if err != nil || strings.TrimSpace(string(owner)) == "" {
		return fmt.Errorf("owner token unavailable: setup_required=%t; netbird/owner-token missing", instance.SetupRequired)
	}
	ownerToken := strings.TrimSpace(string(owner))
	raw, err := call(client, http.MethodGet, api+"/accounts", ownerToken, nil)
	if err != nil {
		return err
	}
	var accounts []struct {
		ID       string         `json:"id"`
		Settings map[string]any `json:"settings"`
	}
	if json.Unmarshal(raw, &accounts) != nil || len(accounts) != 1 {
		return fmt.Errorf("expected one account")
	}
	account := accounts[0]
	desired := map[string]any{
		"ipv6_enabled_groups": []any{}, "auto_update_version": "disabled",
		"metrics_push_enabled": false, "jwt_groups_enabled": false, "groups_propagation_enabled": false,
	}
	changed := false
	for k, v := range desired {
		if fmt.Sprint(account.Settings[k]) != fmt.Sprint(v) {
			changed = true
		}
		account.Settings[k] = v
	}
	if changed {
		if _, err := call(client, http.MethodPut, api+"/accounts/"+account.ID, ownerToken, map[string]any{"settings": account.Settings}); err != nil {
			return err
		}
	}
	raw, err = call(client, http.MethodGet, api+"/groups", ownerToken, nil)
	if err != nil {
		return err
	}
	var groups []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if json.Unmarshal(raw, &groups) != nil {
		return fmt.Errorf("groups")
	}
	allID := ""
	for _, g := range groups {
		if g.Name == "All" {
			allID = g.ID
		}
	}
	raw, err = call(client, http.MethodGet, api+"/policies", ownerToken, nil)
	if err != nil {
		return err
	}
	var policies []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Rules []struct {
			Sources []struct {
				ID string `json:"id"`
			} `json:"sources"`
			Destinations []struct {
				ID string `json:"id"`
			} `json:"destinations"`
		} `json:"rules"`
	}
	if json.Unmarshal(raw, &policies) != nil {
		return fmt.Errorf("policies")
	}
	for _, policy := range policies {
		broad := false
		for _, rule := range policy.Rules {
			for _, g := range rule.Sources {
				if g.ID == allID && allID != "" {
					broad = true
				}
			}
			for _, g := range rule.Destinations {
				if g.ID == allID && allID != "" {
					broad = true
				}
			}
		}
		if broad {
			if policy.Name != "Default" {
				return fmt.Errorf("unexpected broad policy")
			}
			if _, err := call(client, http.MethodDelete, api+"/policies/"+policy.ID, ownerToken, nil); err != nil {
				return err
			}
		}
	}
	raw, err = call(client, http.MethodGet, api+"/users", ownerToken, nil)
	if err != nil {
		return err
	}
	var users []struct {
		ID            string   `json:"id"`
		Name          string   `json:"name"`
		Email         string   `json:"email"`
		Role          string   `json:"role"`
		IsServiceUser bool     `json:"is_service_user"`
		IsBlocked     bool     `json:"is_blocked"`
		AutoGroups    []string `json:"auto_groups"`
	}
	if json.Unmarshal(raw, &users) != nil {
		return fmt.Errorf("users")
	}
	var serviceID string
	for _, u := range users {
		if u.Name == "fvpn-adapter" && u.IsServiceUser {
			if serviceID != "" {
				return fmt.Errorf("duplicate adapter")
			}
			if u.Role != "admin" || len(u.AutoGroups) != 0 {
				return fmt.Errorf("unexpected adapter permissions")
			}
			serviceID = u.ID
		}
	}
	if serviceID == "" {
		raw, err = call(client, http.MethodPost, api+"/users", ownerToken, map[string]any{
			"name": "fvpn-adapter", "role": "admin", "is_service_user": true, "auto_groups": []any{},
		})
		if err != nil {
			return err
		}
		var created struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &created) != nil || created.ID == "" {
			return fmt.Errorf("adapter user")
		}
		serviceID = created.ID
	}
	tokenPath := filepath.Join(state, "secrets", "netbird.token")
	if strings.TrimSpace(mustRead(tokenPath)) == "pending-netbird-bootstrap" || strings.TrimSpace(mustRead(tokenPath)) == "" {
		path := api + "/users/" + serviceID + "/tokens"
		raw, err = call(client, http.MethodGet, path, ownerToken, nil)
		if err != nil {
			return err
		}
		var tokens []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		_ = json.Unmarshal(raw, &tokens)
		for _, token := range tokens {
			if token.Name == "fvpn-adapter-e2e" {
				if _, err := call(client, http.MethodDelete, path+"/"+token.ID, ownerToken, nil); err != nil {
					return err
				}
			}
		}
		raw, err = call(client, http.MethodPost, path, ownerToken, map[string]any{"name": "fvpn-adapter-e2e", "expires_in": 1})
		if err != nil {
			return err
		}
		var pat struct {
			PlainToken string `json:"plain_token"`
		}
		if json.Unmarshal(raw, &pat) != nil || pat.PlainToken == "" {
			return errors.New(httpFailure(http.StatusOK, raw, "plain_token missing"))
		}
		if err := writeSecret(tokenPath, pat.PlainToken); err != nil {
			return err
		}
	}
	adapter := strings.TrimSpace(mustRead(tokenPath))
	if _, err := call(client, http.MethodGet, api+"/groups", adapter, nil); err != nil {
		return err
	}
	return ensureMemberUser(client, api, ownerToken, nb)
}

// ensureMemberUser creates two ordinary embedded-IdP users. The acceptance
// member signs in as the first. The second is addressable by the harness
// key e2e-member-b. Passwords stay in the state directory.
func ensureMemberUser(client *http.Client, api, ownerToken, nb string) error {
	users := []struct{ email, name, file string }{
		{"e2e-member@example.invalid", "e2e-member", "member-user.json"},
		{"e2e-member-b@example.invalid", "e2e-member-b", "member-user-b.json"},
	}
	for _, user := range users {
		if err := ensurePasswordUser(client, api, ownerToken, nb, user.email, user.name, user.file); err != nil {
			return err
		}
	}
	return nil
}

func ensurePasswordUser(client *http.Client, api, ownerToken, nb, email, name, file string) error {
	raw, err := call(client, http.MethodGet, api+"/users", ownerToken, nil)
	if err != nil {
		return err
	}
	var users []struct {
		ID            string   `json:"id"`
		Name          string   `json:"name"`
		Email         string   `json:"email"`
		Role          string   `json:"role"`
		IsServiceUser bool     `json:"is_service_user"`
		IsBlocked     bool     `json:"is_blocked"`
		AutoGroups    []string `json:"auto_groups"`
	}
	if json.Unmarshal(raw, &users) != nil {
		return fmt.Errorf("users")
	}
	path := filepath.Join(nb, file)
	found := ""
	for _, u := range users {
		if u.Email != email && u.Name != name {
			continue
		}
		if found != "" {
			return fmt.Errorf("duplicate member user")
		}
		if u.IsServiceUser || u.IsBlocked || u.Role != "user" || len(u.AutoGroups) != 0 {
			return fmt.Errorf("unexpected member user permissions")
		}
		found = u.ID
	}
	if found != "" {
		if strings.TrimSpace(mustRead(path)) == "" {
			return fmt.Errorf("member user password unavailable; reset the lab")
		}
		return nil
	}
	raw, err = call(client, http.MethodPost, api+"/users", ownerToken, map[string]any{
		"name": name, "email": email, "role": "user", "is_service_user": false, "auto_groups": []any{},
	})
	if err != nil {
		return err
	}
	var created struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if json.Unmarshal(raw, &created) != nil || created.ID == "" || created.Password == "" {
		return errors.New("member user password missing")
	}
	emailOut := created.Email
	if emailOut == "" {
		emailOut = email
	}
	saved, err := json.Marshal(map[string]string{"id": created.ID, "email": emailOut, "password": created.Password})
	if err != nil {
		return err
	}
	return writeSecret(path, string(saved))
}

func mustRead(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func writeSecret(path, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(value)+"\n"), 0o600)
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	out := make([]byte, len(b))
	for i, v := range b {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(out), nil
}
