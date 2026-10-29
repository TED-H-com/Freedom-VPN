package worker

import "github.com/jackc/pgx/v5/pgxpool"

// Jobs is the single composition point for domain jobs (§14.5).
// Each domain adds one Register line. pool is the worker database.
func Jobs(pool *pgxpool.Pool) Registry {
	r := make(Registry)
	r.Register(dnsPublishEvent, dnsPublishHandler(pool))
	return r
}
