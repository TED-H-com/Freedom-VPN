package worker

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	dnsstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
)

// dns registers DNS publication. policy.activated stays unregistered so
// desired-state compilation can consume it (§14.5). Resolver addresses come
// from Dependencies.Config.DNSConfigFile. There is no production default (D-028).
// The build only constructs the handler; it does not start work.
func dns() Domain {
	return Domain{Name: "dns", Build: func(_ context.Context, d Dependencies) (Built, error) {
		dnsCfg, err := config.LoadDNS(d.Config.DNSConfigFile)
		if err != nil {
			return Built{}, &config.SettingError{Name: "dns_config_file", Err: err}
		}
		publisher, err := dnsstore.NewPublisher(d.Pool, dnsstore.Config{
			Upstreams:    dnsCfg.Upstreams,
			BootstrapDNS: dnsCfg.BootstrapDNS,
		})
		if err != nil {
			return Built{}, &config.SettingError{Name: "dns_config_file", Err: err}
		}
		jobs := Registry{}
		jobs.Register(dnspublish.EventDNSPublish, publisher.Handle)
		return Built{Jobs: jobs, Runners: nil}, nil
	}}
}
