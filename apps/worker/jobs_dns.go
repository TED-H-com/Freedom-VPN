package worker

import (
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/jackc/pgx/v5/pgxpool"
)

// dnsPublishEvent is fanned out from dns_profile activation. policy.activated
// stays unregistered here so desired-state compilation can consume it (§14.5).
const dnsPublishEvent = dnspublish.EventDNSPublish

func dnsPublishHandler(pool *pgxpool.Pool) Handler {
	return dnspublish.NewHandler(pool, dnspublish.DefaultConfig())
}
