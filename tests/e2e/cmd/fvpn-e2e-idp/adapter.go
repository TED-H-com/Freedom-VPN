package main

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	errNoKey    = errors.New("no key")
	errRejected = errors.New("rejected")
)

func (s *server) oauthToken(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		oauthErr(w, http.StatusBadRequest, "invalid_request", "bad body")
		return
	}
	values, err := url.ParseQuery(string(raw))
	if err != nil {
		s.proxyDex(w, r, raw, true)
		return
	}
	id, _, ok := r.BasicAuth()
	if !ok {
		id = values.Get("client_id")
	}
	portal := values.Get("grant_type") == "authorization_code" && id == s.portalClient && values.Get("redirect_uri") == s.redirect
	if !portal {
		s.proxyDex(w, r, raw, true)
		return
	}
	forward := url.Values{}
	forward.Set("grant_type", "authorization_code")
	forward.Set("code", values.Get("code"))
	forward.Set("redirect_uri", values.Get("redirect_uri"))
	forward.Set("client_id", id)
	forward.Set("code_verifier", values.Get("code_verifier"))
	status, body, err := s.postDex(r, forward.Encode(), false)
	if err != nil {
		oauthErr(w, http.StatusBadGateway, "temporarily_unavailable", "token upstream unavailable")
		return
	}
	if status != http.StatusOK {
		writeRaw(w, status, body)
		return
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		oauthErr(w, http.StatusBadGateway, "temporarily_unavailable", "token upstream rejected")
		return
	}
	dexToken, _ := payload["id_token"].(string)
	resigned, err := s.resignPortal(r, dexToken)
	if err != nil {
		oauthErr(w, http.StatusBadGateway, "invalid_grant", "portal token rejected")
		return
	}
	payload["id_token"] = resigned
	writeJSON(w, http.StatusOK, payload)
}

func (s *server) oauthKeys(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.dexUpstream+"/oauth2/keys", nil)
	if err != nil {
		oauthErr(w, http.StatusBadGateway, "temporarily_unavailable", "jwks unavailable")
		return
	}
	s.decorateDex(req)
	resp, err := s.dexClient.Do(req)
	if err != nil {
		oauthErr(w, http.StatusBadGateway, "temporarily_unavailable", "jwks unavailable")
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || resp.StatusCode != http.StatusOK {
		oauthErr(w, http.StatusBadGateway, "temporarily_unavailable", "jwks unavailable")
		return
	}
	var doc struct {
		Keys []map[string]any `json:"keys"`
	}
	if json.Unmarshal(body, &doc) != nil {
		oauthErr(w, http.StatusBadGateway, "temporarily_unavailable", "jwks unavailable")
		return
	}
	doc.Keys = append(doc.Keys, jwkAny(s.publicJWK(s.portalKid)))
	writeJSON(w, http.StatusOK, map[string]any{"keys": doc.Keys})
}

func (s *server) resignPortal(r *http.Request, dexToken string) (string, error) {
	keys, err := s.dexPublicKeys(r)
	if err != nil {
		return "", err
	}
	claims, err := verifyRS256(dexToken, func(kid string) (*rsa.PublicKey, error) {
		pub, ok := keys[kid]
		if !ok {
			return nil, errNoKey
		}
		return pub, nil
	})
	if err != nil {
		return "", err
	}
	if claimString(claims["iss"]) != s.embedIssuer || claimString(claims["sub"]) == "" || !audienceMatches(claims["aud"], s.portalClient) {
		return "", errRejected
	}
	now := time.Now().Unix()
	exp := claimInt(claims["exp"])
	iat := claimInt(claims["iat"])
	if exp <= now || iat == 0 {
		return "", errRejected
	}
	authTime := claimInt(claims["auth_time"])
	if authTime == 0 {
		authTime = iat
	}
	name := claimString(claims["name"])
	email := claimString(claims["email"])
	preferred := claimString(claims["preferred_username"])
	roles := []string{"member"}
	if s.isAdmin(name, email, preferred) {
		roles = []string{"admin"}
	}
	out := map[string]any{
		"iss": s.embedIssuer, "sub": claimString(claims["sub"]), "aud": s.portalClient,
		"iat": iat, "exp": exp, "auth_time": authTime, "nonce": claimString(claims["nonce"]),
		"amr": []string{"mfa"}, "roles": roles,
	}
	if name != "" {
		out["name"] = name
	}
	if at := claimString(claims["at_hash"]); at != "" {
		out["at_hash"] = at
	}
	return signRS256(s.key, s.portalKid, out)
}

func (s *server) isAdmin(name, email, preferred string) bool {
	return sameText(email, s.adminEmail) || sameText(name, s.adminName) || sameText(preferred, s.adminEmail) || sameText(name, s.adminEmail) || sameText(preferred, s.adminName)
}

func sameText(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	return a != "" && strings.EqualFold(a, b)
}

func (s *server) dexPublicKeys(r *http.Request) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.dexUpstream+"/oauth2/keys", nil)
	if err != nil {
		return nil, err
	}
	s.decorateDex(req)
	resp, err := s.dexClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errRejected
	}
	var doc struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return nil, errRejected
	}
	out := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		pub, err := rsaFromModExp(k.N, k.E)
		if err != nil || k.Kid == "" {
			continue
		}
		out[k.Kid] = pub
	}
	if len(out) == 0 {
		return nil, errNoKey
	}
	return out, nil
}

func (s *server) postDex(r *http.Request, body string, forwardAuth bool) (int, []byte, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.dexUpstream+"/oauth2/token", strings.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if forwardAuth {
		if auth := r.Header.Get("Authorization"); auth != "" {
			req.Header.Set("Authorization", auth)
		}
	}
	s.decorateDex(req)
	resp, err := s.dexClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, raw, nil
}

func (s *server) proxyDex(w http.ResponseWriter, r *http.Request, raw []byte, forwardAuth bool) {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/x-www-form-urlencoded"
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.dexUpstream+"/oauth2/token", bytes.NewReader(raw))
	if err != nil {
		oauthErr(w, http.StatusBadGateway, "temporarily_unavailable", "token upstream unavailable")
		return
	}
	req.Header.Set("Content-Type", contentType)
	if forwardAuth {
		if auth := r.Header.Get("Authorization"); auth != "" {
			req.Header.Set("Authorization", auth)
		}
	}
	s.decorateDex(req)
	resp, err := s.dexClient.Do(req)
	if err != nil {
		oauthErr(w, http.StatusBadGateway, "temporarily_unavailable", "token upstream unavailable")
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		oauthErr(w, http.StatusBadGateway, "temporarily_unavailable", "token upstream unavailable")
		return
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (s *server) decorateDex(req *http.Request) {
	if s.dexHost != "" {
		req.Host = s.dexHost
	}
}

func oauthErr(w http.ResponseWriter, status int, code, desc string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": desc})
}

func writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func jwkAny(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
