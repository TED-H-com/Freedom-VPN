//go:build e2e

package members

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Supplemental, observation-only privacy inventory for the second exit.
// Query samples are synthetic and only aggregate match counts leave the lab.
func (l *live) secondPrivacy(r *record) {
	if _, exists := l.build.Images["node2-adguard"]; !exists {
		r.block("Second resolver privacy inventory unavailable")
		return
	}
	sentinel := "privacy-secondary-" + l.run + ".example.invalid"
	answer, e := l.inside("node2", "/tmp/at-members-probe", "dns", "127.0.0.1:53", sentinel, "1", "udp")
	r.check("second resolver privacy sample", "controlled DNS response", string(answer), "node2 loopback synthetic DNS sample", e == nil && bytes.Contains(answer, []byte(`"available":true`)))
	raw, e := l.inside("node2-adguard", "cat", "/opt/adguardhome/conf/AdGuardHome.yaml")
	var cfg struct {
		Querylog struct {
			Enabled     bool
			FileEnabled bool `yaml:"file_enabled"`
		}
		Statistics struct{ Enabled bool }
	}
	decoded := yaml.Unmarshal(raw, &cfg)
	r.check("second resolver on-disk privacy", "query log/file and statistics disabled", fmt.Sprintf("read_ok=%t; querylog=%t; file=%t; statistics=%t", e == nil && decoded == nil, cfg.Querylog.Enabled, cfg.Querylog.FileEnabled, cfg.Statistics.Enabled), "parse second resolver config in memory; no credentials retained", e == nil && decoded == nil && !cfg.Querylog.Enabled && !cfg.Querylog.FileEnabled && !cfg.Statistics.Enabled)
	for _, name := range []string{"node2", "node2-adguard"} {
		raw, e := l.docker("logs", l.name(name))
		count := bytes.Count(raw, []byte(sentinel))
		r.check(name+" service log privacy", "zero synthetic qname matches", fmt.Sprintf("read_ok=%t; matches=%d", e == nil, count), "scan assigned secondary service logs in memory", e == nil && count == 0)
	}
}
