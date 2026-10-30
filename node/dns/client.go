package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const bodyLimit = 4 << 20

type engines struct {
	Bing       bool `json:"bing"`
	DuckDuckGo bool `json:"duckduckgo"`
	Ecosia     bool `json:"ecosia"`
	Google     bool `json:"google"`
	Pixabay    bool `json:"pixabay"`
	Yandex     bool `json:"yandex"`
	YouTube    bool `json:"youtube"`
}

type resolverState struct {
	Rules        []string
	Upstreams    []string
	Bootstrap    []string
	Fallback     []string
	BlockingMode string
	BlockedTTL   uint32
	Protection   bool
	Filtering    bool
	QueryLog     bool
	QueryLogFile *bool
	Statistics   bool
	SafeEnabled  bool
	Safe         engines
	Filters      int
}

type safeStatus struct {
	Enabled bool `json:"enabled"`
	engines
}

func (a *Applier) readState(ctx context.Context, t Target) (resolverState, error) {
	var dnsInfo struct {
		UpstreamDNS        []string `json:"upstream_dns"`
		BootstrapDNS       []string `json:"bootstrap_dns"`
		FallbackDNS        []string `json:"fallback_dns"`
		BlockingMode       string   `json:"blocking_mode"`
		BlockedResponseTTL uint32   `json:"blocked_response_ttl"`
		ProtectionEnabled  bool     `json:"protection_enabled"`
	}
	if err := a.call(ctx, t, http.MethodGet, "/control/dns_info", nil, &dnsInfo); err != nil {
		return resolverState{}, err
	}
	var filter struct {
		Enabled bool `json:"enabled"`
		Filters []struct {
			URL     string `json:"url"`
			Enabled bool   `json:"enabled"`
		} `json:"filters"`
		UserRules []string `json:"user_rules"`
	}
	if err := a.call(ctx, t, http.MethodGet, "/control/filtering/status", nil, &filter); err != nil {
		return resolverState{}, err
	}
	var ql struct {
		Enabled     bool  `json:"enabled"`
		FileEnabled *bool `json:"file_enabled"`
	}
	if err := a.call(ctx, t, http.MethodGet, "/control/querylog/config", nil, &ql); err != nil {
		return resolverState{}, err
	}
	var stats struct {
		Enabled bool `json:"enabled"`
	}
	if err := a.call(ctx, t, http.MethodGet, "/control/stats/config", nil, &stats); err != nil {
		return resolverState{}, err
	}
	var safe safeStatus
	if err := a.call(ctx, t, http.MethodGet, "/control/safesearch/status", nil, &safe); err != nil {
		return resolverState{}, err
	}
	enabledFilters := 0
	for _, f := range filter.Filters {
		if f.Enabled {
			enabledFilters++
		}
	}
	return resolverState{
		Rules:        append([]string(nil), filter.UserRules...),
		Upstreams:    append([]string(nil), dnsInfo.UpstreamDNS...),
		Bootstrap:    append([]string(nil), dnsInfo.BootstrapDNS...),
		Fallback:     append([]string(nil), dnsInfo.FallbackDNS...),
		BlockingMode: dnsInfo.BlockingMode,
		BlockedTTL:   dnsInfo.BlockedResponseTTL,
		Protection:   dnsInfo.ProtectionEnabled,
		Filtering:    filter.Enabled,
		QueryLog:     ql.Enabled,
		QueryLogFile: ql.FileEnabled,
		Statistics:   stats.Enabled,
		SafeEnabled:  safe.Enabled,
		Safe:         safe.engines,
		Filters:      enabledFilters,
	}, nil
}

func (a *Applier) writeState(ctx context.Context, t Target, st resolverState) error {
	ql := map[string]any{
		"enabled":             st.QueryLog,
		"file_enabled":        false,
		"interval":            86400000,
		"anonymize_client_ip": true,
		"ignored":             []string{},
	}
	if st.QueryLogFile != nil {
		ql["file_enabled"] = *st.QueryLogFile
	}
	if err := a.call(ctx, t, http.MethodPut, "/control/querylog/config/update", ql, nil); err != nil {
		return fmt.Errorf("%s: disable query log: %w", t.ID, err)
	}
	if err := a.call(ctx, t, http.MethodPut, "/control/stats/config/update", map[string]any{
		"enabled":  st.Statistics,
		"interval": 86400000,
		"ignored":  []string{},
	}, nil); err != nil {
		return fmt.Errorf("%s: disable statistics: %w", t.ID, err)
	}
	if err := a.call(ctx, t, http.MethodPut, "/control/safesearch/settings", safeStatus{Enabled: st.SafeEnabled, engines: st.Safe}, nil); err != nil {
		return fmt.Errorf("%s: safe search: %w", t.ID, err)
	}
	if err := a.removeFilters(ctx, t); err != nil {
		return err
	}
	if err := a.call(ctx, t, http.MethodPost, "/control/filtering/config", map[string]any{
		"enabled": st.Filtering, "interval": 0,
	}, nil); err != nil {
		return fmt.Errorf("%s: filtering: %w", t.ID, err)
	}
	bootstrap := st.Bootstrap
	if len(bootstrap) == 0 {
		bootstrap = st.Upstreams
	}
	fallback := st.Fallback
	if fallback == nil {
		fallback = []string{}
	}
	if err := a.call(ctx, t, http.MethodPost, "/control/dns_config", map[string]any{
		"upstream_dns":              st.Upstreams,
		"bootstrap_dns":             bootstrap,
		"fallback_dns":              fallback,
		"protection_enabled":        st.Protection,
		"ratelimit":                 0,
		"blocking_mode":             st.BlockingMode,
		"blocked_response_ttl":      st.BlockedTTL,
		"edns_cs_enabled":           false,
		"dnssec_enabled":            false,
		"disable_ipv6":              false,
		"use_private_ptr_resolvers": false,
		"local_ptr_upstreams":       []string{},
	}, nil); err != nil {
		return fmt.Errorf("%s: dns config: %w", t.ID, err)
	}
	rules := st.Rules
	if rules == nil {
		rules = []string{}
	}
	if err := a.call(ctx, t, http.MethodPost, "/control/filtering/set_rules", map[string]any{"rules": rules}, nil); err != nil {
		return fmt.Errorf("%s: rules: %w", t.ID, err)
	}
	if err := a.call(ctx, t, http.MethodPost, "/control/cache_clear", map[string]any{}, nil); err != nil {
		return fmt.Errorf("%s: cache: %w", t.ID, err)
	}
	return nil
}

func (a *Applier) removeFilters(ctx context.Context, t Target) error {
	var filter struct {
		Filters []struct {
			URL string `json:"url"`
		} `json:"filters"`
	}
	if err := a.call(ctx, t, http.MethodGet, "/control/filtering/status", nil, &filter); err != nil {
		return err
	}
	for _, f := range filter.Filters {
		if strings.TrimSpace(f.URL) == "" {
			continue
		}
		if err := a.call(ctx, t, http.MethodPost, "/control/filtering/remove_url", map[string]any{
			"url": f.URL, "whitelist": false,
		}, nil); err != nil {
			return fmt.Errorf("%s: remove external filter: %w", t.ID, err)
		}
	}
	return nil
}

func (a *Applier) call(ctx context.Context, t Target, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(t.AdminURL, "/")+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if t.Username != "" || t.Password != "" {
		req.SetBasicAuth(t.Username, t.Password)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		if errors.Is(err, errAdminRedirect) {
			// net/http attaches the redirect Location to this error. Keep the
			// same status-and-path sentence as any other non-2xx (§16.4).
			code := 0
			if resp != nil {
				code = resp.StatusCode
			}
			return fmt.Errorf("HTTP %d from %s %s", code, method, path)
		}
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, bodyLimit))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// The response body can echo configuration. Keep the status and path only (§16.4).
		return fmt.Errorf("HTTP %d from %s %s", resp.StatusCode, method, path)
	}
	if out != nil && len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode %s %s: %w", method, path, err)
		}
	}
	return nil
}

// errAdminRedirect is the CheckRedirect sentinel. call replaces the wrapped
// error, which carries the Location, with the fixed HTTP sentence.
var errAdminRedirect = errors.New("dns: redirect refused")

// refuseAdminRedirect rejects every redirect, including a same-host port
// change. That matches node/internal/controlclient refuseRedirect. Both the
// load path and the SIGHUP reload path build the applier through newHTTPClient.
func refuseAdminRedirect(*http.Request, []*http.Request) error {
	return errAdminRedirect
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second, CheckRedirect: refuseAdminRedirect}
}
