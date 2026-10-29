package worker

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	dnsstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
)

// dns registers DNS publication. policy.activated stays unregistered so
// desired-state compilation can consume it (§14.5). The build only constructs
// the handler; it does not start work.
func dns() Domain {
	return Domain{Name: "dns", Build: func(_ context.Context, d Dependencies) (Built, error) {
		publisher := dnsstore.NewPublisher(d.Pool, dnsstore.DefaultConfig())
		jobs := Registry{}
		jobs.Register(dnspublish.EventDNSPublish, publisher.Handle)
		return Built{Jobs: jobs, Runners: nil}, nil
	}}
}
