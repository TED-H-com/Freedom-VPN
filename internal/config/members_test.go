package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func validMemberSettings() Members {
	zero := 0
	live := false
	return Members{Provider: "stripe", APIKeyFile: "TEST-ONLY-api-key", WebhookSecretFiles: []string{"TEST-ONLY-webhook"}, VaultActiveKey: "TEST-ONLY", VaultKeyFiles: map[string]string{"TEST-ONLY": "TEST-ONLY-vault-key"}, LiveMode: &live, PriceIDs: []string{"TEST-ONLY-price"}, ReturnURLs: []string{"https://portal.example.invalid/billing/return"}, FullRefundPolicy: "revoke_original_period", GraceSeconds: &zero, TrialMaxSeconds: &zero, GrantMaxSeconds: 86400, DeviceLimit: 3, PortalTTLSeconds: 60, BillingLeaseSeconds: 60, MaxWebhookBytes: 1 << 20, ReconcileSeconds: 300, DeviceSyncSeconds: 30, NetBirdURL: "https://netbird.example.invalid/api", NetBirdDeployment: "TEST-ONLY", NetBirdTokenFile: "TEST-ONLY-netbird-key", OverlayScope: "TEST-ONLY"}
}
func TestBILL01_ExplicitBusinessSettingsAndRedactedConfiguration(t *testing.T) {
	c := validMemberSettings()
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*Members){func(c *Members) { c.DeviceLimit = 0 }, func(c *Members) { c.GraceSeconds = nil }, func(c *Members) { c.TrialMaxSeconds = nil }, func(c *Members) { c.PriceIDs = nil }, func(c *Members) { c.LiveMode = nil }, func(c *Members) { c.FullRefundPolicy = "" }, func(c *Members) { c.NetBirdURL = "http://netbird.example.invalid/api" }} {
		bad := c
		mutate(&bad)
		if bad.Validate() == nil {
			t.Fatal("missing/unsafe business setting accepted")
		}
	}
	var logs bytes.Buffer
	slog.New(slog.NewJSONHandler(&logs, nil)).Info("config", "members", c)
	for _, v := range []string{c.APIKeyFile, c.NetBirdURL, c.NetBirdTokenFile, c.VaultKeyFiles[c.VaultActiveKey]} {
		if strings.Contains(logs.String(), v) {
			t.Fatal("configuration leaked secret location")
		}
	}
	if got := Defaults().MembersConfigFile; got != "" {
		t.Fatal("unexpected implicit member configuration")
	}
}
