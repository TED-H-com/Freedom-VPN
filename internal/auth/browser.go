package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

const AdminSessionCookie = "__Host-fvpn-admin"
const MemberSessionCookie = "__Host-fvpn-member"
const flowCookie = "__Host-fvpn-login"

type BrowserConfig struct {
	Issuer, ClientID, ClientSecret, Origin, AdminClaim, AdminValue string
	MFAAMR, MFAACR                                                 []string
	Idle, Absolute, LoginTTL, RecentAuth                           time.Duration
}
type Browser struct {
	config   BrowserConfig
	store    Store
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
	client   *http.Client
}

func NewBrowser(ctx context.Context, c BrowserConfig, s Store, client *http.Client) (*Browser, error) {
	if c.Issuer == "" || c.ClientID == "" || c.Origin == "" || s == nil || c.Idle <= 0 || c.Absolute < c.Idle || c.LoginTTL <= 0 || c.RecentAuth <= 0 || c.AdminClaim == "" || c.AdminValue == "" || len(c.MFAAMR)+len(c.MFAACR) == 0 {
		return nil, errors.New("invalid browser authentication configuration")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	provider, e := oidc.NewProvider(oidc.ClientContext(ctx, client), c.Issuer)
	if e != nil {
		return nil, errors.New("OIDC discovery failed")
	}
	return &Browser{config: c, store: s, client: client, oauth: oauth2.Config{ClientID: c.ClientID, ClientSecret: c.ClientSecret, Endpoint: provider.Endpoint(), RedirectURL: c.Origin + "/auth/callback", Scopes: []string{oidc.ScopeOpenID, "profile"}}, verifier: provider.Verifier(&oidc.Config{ClientID: c.ClientID})}, nil
}
func randomToken() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func cookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func cookieValue(r *http.Request, name string) string {
	c, e := r.Cookie(name)
	if e != nil {
		return ""
	}
	return c.Value
}
func csrf(token string) string {
	h := hmac.New(sha256.New, []byte(token))
	_, _ = h.Write([]byte("fvpn csrf v1"))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func CheckCSRF(r *http.Request, p Principal, origin string) error {
	v := r.Header.Get("X-CSRF-Token")
	if r.Header.Get("Origin") != origin || p.CSRFToken == "" || subtle.ConstantTimeCompare([]byte(v), []byte(p.CSRFToken)) != 1 {
		return apierr.New(apierr.CSRFTokenInvalid, "Invalid CSRF token or origin.")
	}
	return nil
}
func (b *Browser) AuthenticateSession(r *http.Request) (Principal, error) {
	role, e := sessionRole(r)
	if e != nil {
		return Principal{}, e
	}
	token := cookieValue(r, cookieNameForRole(role))
	if token == "" {
		return Principal{}, apierr.New(apierr.Unauthenticated, "Sign in required.")
	}
	p, e := b.store.Session(r.Context(), token, b.config.Idle)
	if e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return p, apierr.New(apierr.Unauthenticated, "Session expired.")
		}
		return p, apierr.New(apierr.DependencyUnavailable, "Session service unavailable.")
	}
	if p.Role != role {
		return Principal{}, apierr.New(apierr.Unauthenticated, "Session role mismatch.")
	}
	p.CSRFToken = csrf(token)
	return p, nil
}
func (b *Browser) Register(m *http.ServeMux) {
	m.HandleFunc("GET /auth/login", b.login)
	m.HandleFunc("GET /auth/callback", b.callback)
	m.HandleFunc("POST /auth/logout", b.logout)
	m.HandleFunc("GET /auth/session", b.session)
}
func (b *Browser) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	f := Flow{State: randomToken(), Browser: randomToken(), Nonce: randomToken(), Verifier: oauth2.GenerateVerifier(), Expires: time.Now().UTC().Add(b.config.LoginTTL)}
	if e := b.store.SaveFlow(r.Context(), f); e != nil {
		apierr.Write(w, r, apierr.New(apierr.DependencyUnavailable, "Login unavailable."))
		return
	}
	cookie(w, flowCookie, f.Browser, int(b.config.LoginTTL.Seconds()))
	http.Redirect(w, r, b.oauth.AuthCodeURL(f.State, oauth2.S256ChallengeOption(f.Verifier), oidc.Nonce(f.Nonce), oauth2.SetAuthURLParam("max_age", "0")), http.StatusFound)
}
func (b *Browser) callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	fail := func() { apierr.Write(w, r, apierr.New(apierr.Unauthenticated, "Login verification failed.")) }
	state := r.URL.Query().Get("state")
	browser := cookieValue(r, flowCookie)
	cookie(w, flowCookie, "", -1)
	if state == "" || browser == "" || r.URL.Query().Get("code") == "" {
		fail()
		return
	}
	f, e := b.store.ConsumeFlow(r.Context(), state, browser)
	if e != nil {
		fail()
		return
	}
	ctx := oidc.ClientContext(r.Context(), b.client)
	token, e := b.oauth.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(f.Verifier))
	if e != nil {
		fail()
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		fail()
		return
	}
	id, e := b.verifier.Verify(ctx, raw)
	if e != nil || id.Subject == "" || subtle.ConstantTimeCompare([]byte(id.Nonce), []byte(f.Nonce)) != 1 {
		fail()
		return
	}
	var claims map[string]json.RawMessage
	if id.Claims(&claims) != nil {
		fail()
		return
	}
	var amr []string
	var acr, label string
	var authTime int64
	_ = json.Unmarshal(claims["amr"], &amr)
	_ = json.Unmarshal(claims["acr"], &acr)
	_ = json.Unmarshal(claims["name"], &label)
	_ = json.Unmarshal(claims["auth_time"], &authTime)
	role := Member
	var roles []string
	var roleString string
	_ = json.Unmarshal(claims[b.config.AdminClaim], &roles)
	_ = json.Unmarshal(claims[b.config.AdminClaim], &roleString)
	if roleString == b.config.AdminValue || slices.Contains(roles, b.config.AdminValue) {
		role = Admin
	}
	mfa := acr != "" && slices.Contains(b.config.MFAACR, acr)
	for _, a := range amr {
		if slices.Contains(b.config.MFAAMR, a) {
			mfa = true
		}
	}
	at := time.Unix(authTime, 0).UTC()
	if authTime == 0 {
		if role == Admin {
			fail()
			return
		}
		at = id.IssuedAt
	}
	if at.After(time.Now().Add(time.Minute)) {
		fail()
		return
	}
	if role == Admin && !mfa {
		apierr.Write(w, r, apierr.New(apierr.Forbidden, "Administrator MFA required."))
		return
	}
	if len(label) > 200 {
		label = ""
	}
	if label == "" {
		label = string(role)
	}
	newToken := randomToken()
	rid, _ := apierr.RequestIDFromContext(r.Context())
	requestID, _ := uuid.Parse(rid)
	if requestID == uuid.Nil {
		requestID = uuid.Must(uuid.NewV7())
	}
	_, e = b.store.Login(r.Context(), Identity{Issuer: id.Issuer, ClientScope: b.config.ClientID, Subject: id.Subject, Label: label, Role: role, MFA: mfa, AuthTime: at, Expires: id.Expiry}, newToken, cookieValue(r, cookieNameForRole(role)), b.config.Absolute, requestID)
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	cookie(w, cookieNameForRole(role), newToken, int(b.config.Absolute.Seconds()))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func (b *Browser) logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	p, e := b.AuthenticateSession(r)
	if e == nil {
		e = CheckCSRF(r, p, b.config.Origin)
	}
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	if e = b.store.Logout(r.Context(), cookieValue(r, cookieNameForRole(p.Role))); e != nil {
		apierr.Write(w, r, e)
		return
	}
	cookie(w, cookieNameForRole(p.Role), "", -1)
	w.WriteHeader(http.StatusNoContent)
}
func (b *Browser) session(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	p, e := b.AuthenticateSession(r)
	if errors.Is(e, apierr.New(apierr.Unauthenticated, "")) {
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
		return
	}
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	w.Header().Set("X-CSRF-Token", p.CSRFToken)
	data := map[string]any{"authenticated": true, "role": p.Role, "display_label": p.DisplayLabel, "mfa_verified": p.MFAVerified}
	if p.Role == Member {
		data["member_id"] = p.ID
	}
	_ = json.NewEncoder(w).Encode(data)
}

// API paths select their own role cookie. Browser bootstrap/logout require a role
// when both cookies are present, so an admin session can never shadow a member.
func sessionRole(r *http.Request) (Role, error) {
	if strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
		return Admin, nil
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/member/") {
		return Member, nil
	}
	values, explicit := r.URL.Query()["role"]
	if explicit {
		if len(values) != 1 || (values[0] != string(Admin) && values[0] != string(Member)) {
			return "", apierr.New(apierr.MalformedRequest, "Role must be admin or member.")
		}
		return Role(values[0]), nil
	}
	admin := cookieValue(r, AdminSessionCookie) != ""
	member := cookieValue(r, MemberSessionCookie) != ""
	if admin && member {
		return "", apierr.New(apierr.MalformedRequest, "Select a session role.")
	}
	if admin {
		return Admin, nil
	}
	return Member, nil
}
func cookieNameForRole(role Role) string {
	if role == Admin {
		return AdminSessionCookie
	}
	return MemberSessionCookie
}
