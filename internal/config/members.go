package config

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
)

// Members contains explicit approved business settings. Zero grace/trial means
// disabled only when explicitly supplied; there are no production business defaults.
// The main configuration points to this separately protected JSON file.
type Members struct {
	Provider            string               `json:"provider"`
	APIKeyFile          string               `json:"api_key_file"`
	WebhookSecretFiles  []string             `json:"webhook_secret_files"`
	VaultActiveKey      string               `json:"vault_active_key"`
	VaultKeyFiles       map[string]string    `json:"vault_key_files"`
	LiveMode            *bool                `json:"live_mode"`
	PriceIDs            []string             `json:"price_ids"`
	ReturnURLs          []string             `json:"return_urls"`
	FullRefundPolicy    billing.RefundPolicy `json:"full_refund_policy"`
	GraceSeconds        *int                 `json:"grace_seconds"`
	GraceReason         string               `json:"grace_reason"`
	TrialMaxSeconds     *int                 `json:"trial_max_seconds"`
	GrantMaxSeconds     int                  `json:"grant_max_seconds"`
	DeviceLimit         int                  `json:"device_limit"`
	PortalTTLSeconds    int                  `json:"portal_ttl_seconds"`
	BillingLeaseSeconds int                  `json:"billing_lease_seconds"`
	MaxWebhookBytes     int                  `json:"max_webhook_bytes"`
	ReconcileSeconds    int                  `json:"reconcile_seconds"`
	DeviceSyncSeconds   int                  `json:"device_sync_seconds"`
	NetBirdURL          string               `json:"netbird_url"`
	NetBirdDeployment   string               `json:"netbird_deployment"`
	NetBirdTokenFile    string               `json:"netbird_token_file"`
	OverlayScope        string               `json:"overlay_scope"`
}

func (c Members) Entitlements() (entitlement.Config, error) {
	if c.GraceSeconds == nil || c.TrialMaxSeconds == nil {
		return entitlement.Config{}, errors.New("grace_seconds and trial_max_seconds required")
	}
	out := entitlement.Config{GraceDuration: time.Duration(*c.GraceSeconds) * time.Second, GraceReason: c.GraceReason, MaxTrialDuration: time.Duration(*c.TrialMaxSeconds) * time.Second, MaxGrantDuration: time.Duration(c.GrantMaxSeconds) * time.Second, DefaultDeviceLimit: c.DeviceLimit}
	return out, out.Validate()
}
func (c Members) Validate() error {
	if _, e := c.Entitlements(); e != nil {
		return errors.New("invalid member business settings")
	}
	if c.Provider != "stripe" || c.LiveMode == nil || c.APIKeyFile == "" || len(c.WebhookSecretFiles) == 0 || c.VaultActiveKey == "" || c.VaultKeyFiles[c.VaultActiveKey] == "" || len(c.PriceIDs) == 0 || len(c.ReturnURLs) == 0 || c.NetBirdDeployment == "" || c.NetBirdTokenFile == "" || c.OverlayScope == "" {
		return errors.New("member provider and NetBird settings required")
	}
	if c.FullRefundPolicy != billing.RevokeRefundedPeriod && c.FullRefundPolicy != billing.KeepRefundedPeriod {
		return errors.New("refund policy required")
	}
	if c.PortalTTLSeconds < 1 || c.PortalTTLSeconds > 300 || c.BillingLeaseSeconds < 1 || c.BillingLeaseSeconds > 600 || c.MaxWebhookBytes < 1024 || c.MaxWebhookBytes > 4<<20 || c.ReconcileSeconds < 1 || c.ReconcileSeconds > 86400 || c.DeviceSyncSeconds < 1 || c.DeviceSyncSeconds > 3600 {
		return errors.New("invalid member timing bounds")
	}
	for _, id := range c.PriceIDs {
		if strings.TrimSpace(id) == "" || len(id) > 255 {
			return errors.New("invalid price reference")
		}
	}
	for _, raw := range append(append([]string{}, c.ReturnURLs...), c.NetBirdURL) {
		u, e := url.Parse(raw)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
			return errors.New("member URLs must use HTTPS")
		}
	}
	u, _ := url.Parse(c.NetBirdURL)
	if strings.Trim(u.Path, "/") != "api" || u.RawQuery != "" {
		return errors.New("invalid NetBird API URL")
	}
	return nil
}
func LoadMembers(path string) (Members, error) {
	var c Members
	f, e := os.Open(path)
	if e != nil {
		return c, errors.New("cannot open member configuration")
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return c, errors.New("invalid member configuration")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return c, errors.New("trailing member configuration")
	}
	return c, c.Validate()
}

// LogValue prevents accidental formatting of deployment details and secret paths.
func (c Members) String() string { return "[REDACTED member configuration]" }
