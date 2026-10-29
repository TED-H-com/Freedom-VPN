package control

import (
	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Domain teams add one handler set to this list. A set implements only its own
// generated strict methods, plus Replay for commands. Overlap fails at startup.
func Handlers(pool *pgxpool.Pool) (*httpapi.Composite, error) {
	return httpapi.Compose(
		operations.Store{Pool: pool},
		audit.Store{Pool: pool},
	)
}
