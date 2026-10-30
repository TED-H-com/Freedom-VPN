package worker

import (
	"context"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/dnsfeeds"
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
		feeds, err := dnsfeeds.New(d.Pool, dnsfeeds.Options{UpdateCadence: time.Duration(dnsCfg.FeedUpdateCadenceSeconds) * time.Second, MaxRemovalPercent: dnsCfg.FeedMaxRemovalPercent, Metrics: d.Metrics})
		if err != nil {
			return Built{}, err
		}
		jobs.Register(dnsfeeds.EventRefresh, feeds.Handle)
		return Built{Jobs: jobs, Runners: []func(context.Context) error{func(ctx context.Context) error { return RunDNSFeedSchedule(ctx, feeds) }}}, nil
	}}
}

// RunDNSFeedSchedule follows the worker's supervised periodic runner pattern.
// It repairs missing due intents on startup and every minute; per-source due
// times, durable leases and outbox deduplication are maintained by the service.
func RunDNSFeedSchedule(ctx context.Context, feeds *dnsfeeds.Service) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := feeds.Schedule(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
