package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestReview_F05_OIDCCallbackAcceptsMismatchedAzpAndMultiAudienceTokens proves that
// Browser.callback (/api/v1/auth/callback) checks the OIDC "azp"
// (Authorized Party) claim on both multi-audience ID tokens (such as Dex
// cross-client tokens where aud=["netbird-native-client","portal-client"] and
// azp="netbird-native-client") and single-audience ID tokens carrying a foreign azp.
// Per OIDC Core §3.1.3.7 rules 4 & 5 and D-052 (§9.1, §16.1), a token whose azp
// does not match the portal ClientID must be rejected at /auth/callback.
func TestReview_F05_OIDCCallbackAcceptsMismatchedAzpAndMultiAudienceTokens(t *testing.T) {
	// TEST ONLY - NOT A SECRET: ephemeral RSA key generated at test time.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		aud  any
		azp  any
	}{
		{
			name: "multi_audience_foreign_azp_from_native_client",
			aud:  []string{"netbird-native-client", "test-portal-client"},
			azp:  "netbird-native-client",
		},
		{
			name: "multi_audience_missing_azp",
			aud:  []string{"netbird-native-client", "test-portal-client"},
			azp:  nil,
		},
		{
			name: "single_audience_foreign_azp",
			aud:  "test-portal-client",
			azp:  "netbird-native-client",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var issuer, nonce, challenge string
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/.well-known/openid-configuration":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"issuer":                                issuer,
						"authorization_endpoint":                issuer + "/authorize",
						"token_endpoint":                        issuer + "/token",
						"jwks_uri":                              issuer + "/jwks",
						"id_token_signing_alg_values_supported": []string{"RS256"},
					})
				case "/jwks":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"keys": []any{map[string]any{
							"kty": "RSA",
							"alg": "RS256",
							"use": "sig",
							"kid": "TEST-ONLY",
							"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
							"e":   "AQAB",
						}},
					})
				case "/token":
					_ = r.ParseForm()
					sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
					if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
						return
					}
					now := time.Now()
					claims := map[string]any{
						"iss":       issuer,
						"aud":       tc.aud,
						"sub":       "shared-dex-subject",
						"exp":       now.Add(time.Hour).Unix(),
						"iat":       now.Unix(),
						"auth_time": now.Unix(),
						"nonce":     nonce,
						"name":      "TEST ONLY Member",
					}
					if tc.azp != nil {
						claims["azp"] = tc.azp
					}
					header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"TEST-ONLY"}`))
					body, _ := json.Marshal(claims)
					unsigned := header + "." + base64.RawURLEncoding.EncodeToString(body)
					digest := sha256.Sum256([]byte(unsigned))
					sig, signErr := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
					if signErr != nil {
						t.Error(signErr)
						return
					}
					jwt := unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"access_token": "TEST-ONLY-TOKEN",
						"token_type":   "Bearer",
						"id_token":     jwt,
					})
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer provider.Close()
			issuer = provider.URL

			store := &memoryStore{flows: map[string]Flow{}, sessions: map[string]Principal{}}
			b, err := NewBrowser(context.Background(), BrowserConfig{
				Issuer:     issuer,
				ClientID:   "test-portal-client",
				Origin:     "https://portal.example.invalid",
				AdminClaim: "roles",
				AdminValue: "admin",
				MFAAMR:     []string{"mfa"},
				Idle:       time.Minute,
				Absolute:   time.Hour,
				LoginTTL:   time.Minute,
				RecentAuth: time.Minute,
			}, store, provider.Client())
			if err != nil {
				t.Fatal(err)
			}
			mux := http.NewServeMux()
			b.Register(mux)

			loginRec := httptest.NewRecorder()
			mux.ServeHTTP(loginRec, httptest.NewRequest("GET", "/auth/login", nil))
			if loginRec.Code != http.StatusFound {
				t.Fatalf("login status = %d, want 302", loginRec.Code)
			}
			target, err := url.Parse(loginRec.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			nonce = target.Query().Get("nonce")
			challenge = target.Query().Get("code_challenge")
			state := target.Query().Get("state")

			cbReq := httptest.NewRequest("GET", "/auth/callback?code=test-code&state="+state, nil)
			for _, c := range loginRec.Result().Cookies() {
				cbReq.AddCookie(c)
			}
			cbRec := httptest.NewRecorder()
			mux.ServeHTTP(cbRec, cbReq)

			if cbRec.Code != http.StatusUnauthorized || len(store.identities) != 0 {
				t.Errorf("[F-05]: /auth/callback accepted token with aud=%v azp=%v (HTTP %d, created %d identities); want HTTP 401 and 0 identities",
					tc.aud, tc.azp, cbRec.Code, len(store.identities))
			}
			if len(store.rejections) != 1 || store.rejections[0] != "azp_mismatch" || strings.Contains(store.rejections[0], "TEST-ONLY-TOKEN") {
				t.Errorf("[F-05]: expected 1 token-free azp_mismatch rejection audit, got %v", store.rejections)
			}
		})
	}
}
