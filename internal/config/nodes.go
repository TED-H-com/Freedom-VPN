package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

// Nodes is separately provisioned because PKI lifetimes are business decisions,
// not implicit production defaults (§6.4, §21).
type Nodes struct {
	ProviderDirectory     string   `json:"provider_directory"`
	CACertificateFile     string   `json:"ca_certificate_file"`
	CAKeyID               string   `json:"ca_key_id"`
	CertificateSeconds    int      `json:"certificate_seconds"`
	RenewalOverlapSeconds int      `json:"renewal_overlap_seconds"`
	BootstrapSeconds      int      `json:"bootstrap_seconds"`
	InstallerSeconds      int      `json:"installer_seconds"`
	PreinstalledSeconds   int      `json:"preinstalled_seconds"`
	AuthorizationSeconds  int      `json:"authorization_seconds"`
	OnlineSeconds         int      `json:"online_seconds"`
	StuckSeconds          int      `json:"stuck_seconds"`
	ReconcileSeconds      int      `json:"reconcile_seconds"`
	FetchSeconds          int      `json:"fetch_seconds"`
	DesiredStateSeconds   int      `json:"desired_state_seconds"`
	NetBirdURL            string   `json:"netbird_url"`
	NetBirdDeployment     string   `json:"netbird_deployment"`
	NetBirdTokenFile      string   `json:"netbird_token_file"`
	DNSResolverIPv4       []string `json:"dns_resolver_ipv4"`
	ReleaseOrigins        []string `json:"release_origins"`
}

func LoadNodes() (Nodes, error) {
	c := Nodes{InstallerSeconds: 900, PreinstalledSeconds: 604800}
	f, e := os.Open(os.Getenv("FVPN_NODES_CONFIG_FILE"))
	if e != nil {
		return c, errors.New("node configuration required")
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 65537))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return c, errors.New("invalid node configuration")
	}
	var tail any
	if d.Decode(&tail) != io.EOF {
		return c, errors.New("invalid node configuration trailer")
	}
	return c, c.Validate()
}
func (c Nodes) Validate() error {
	if c.ProviderDirectory == "" || c.CACertificateFile == "" || c.CAKeyID == "" || len(c.ReleaseOrigins) == 0 {
		return errors.New("node authority and release origins required")
	}
	for _, v := range []int{c.CertificateSeconds, c.BootstrapSeconds, c.InstallerSeconds, c.PreinstalledSeconds, c.AuthorizationSeconds, c.OnlineSeconds, c.StuckSeconds, c.ReconcileSeconds, c.FetchSeconds, c.DesiredStateSeconds} {
		if v <= 0 || v > 31536000 {
			return errors.New("invalid node duration")
		}
	}
	if c.RenewalOverlapSeconds < 0 || c.RenewalOverlapSeconds >= c.CertificateSeconds {
		return errors.New("invalid renewal overlap")
	}
	return nil
}
