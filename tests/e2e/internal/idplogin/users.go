package idplogin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Cred is one embedded-IdP user the harness can sign in as.
// Password is the Dex password and must not be logged.
type Cred struct {
	ID       string
	Email    string
	Name     string
	Password string
}

// CredFor returns the lab user for a harness key. e2e-admin, e2e-member and
// e2e-member-b come from bootstrap files. Any other key is created through
// the management API and stored under the state directory.
func CredFor(ctx context.Context, client *http.Client, apiBase, token, stateDir, key string) (Cred, error) {
	switch key {
	case "e2e-admin", "e2e-member", "e2e-member-b":
		return loadKnown(stateDir, key)
	}
	email := safeKey(key) + "@example.invalid"
	path := filepath.Join(stateDir, "netbird", "users", safeKey(key)+".json")
	if cred, err := readCred(path); err == nil {
		return cred, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Cred{}, err
	}
	if client == nil || token == "" {
		return Cred{}, errors.New("lab user unavailable")
	}
	return createUser(ctx, client, apiBase, token, path, email, key)
}

func loadKnown(stateDir, key string) (Cred, error) {
	switch key {
	case "e2e-admin":
		password, err := os.ReadFile(filepath.Join(stateDir, "netbird", "admin-password"))
		if err != nil {
			return Cred{}, err
		}
		if strings.TrimSpace(string(password)) == "" {
			return Cred{}, errors.New("admin password unavailable")
		}
		return Cred{Email: "admin@example.invalid", Name: "E2E Admin", Password: strings.TrimSpace(string(password))}, nil
	case "e2e-member":
		return readCred(filepath.Join(stateDir, "netbird", "member-user.json"))
	case "e2e-member-b":
		return readCred(filepath.Join(stateDir, "netbird", "member-user-b.json"))
	default:
		return Cred{}, os.ErrNotExist
	}
}

func readCred(path string) (Cred, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Cred{}, err
	}
	var cred Cred
	if json.Unmarshal(raw, &cred) != nil || cred.Email == "" || cred.Password == "" {
		return Cred{}, errors.New("lab user unavailable")
	}
	return cred, nil
}

func createUser(ctx context.Context, client *http.Client, apiBase, token, path, email, name string) (Cred, error) {
	apiBase = strings.TrimRight(apiBase, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/users", bytes.NewReader(mustJSON(map[string]any{
		"name": name, "email": email, "role": "user", "is_service_user": false, "auto_groups": []any{},
	})))
	if err != nil {
		return Cred{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+token)
	resp, err := client.Do(req)
	if err != nil {
		return Cred{}, errors.New("netbird user unreachable")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Cred{}, errors.New("netbird user unreadable")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Cred{}, fmt.Errorf("netbird user HTTP %d", resp.StatusCode)
	}
	var created struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if json.Unmarshal(raw, &created) != nil || created.ID == "" || created.Password == "" {
		return Cred{}, errors.New("netbird user password missing")
	}
	if created.Email == "" {
		created.Email = email
	}
	cred := Cred{ID: created.ID, Email: created.Email, Name: name, Password: created.Password}
	saved, err := json.Marshal(map[string]string{"id": cred.ID, "email": cred.Email, "name": cred.Name, "password": cred.Password})
	if err != nil {
		return Cred{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return Cred{}, err
	}
	if err := os.WriteFile(path, append(saved, '\n'), 0o600); err != nil {
		return Cred{}, err
	}
	return cred, nil
}

func safeKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	var b strings.Builder
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "user"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
