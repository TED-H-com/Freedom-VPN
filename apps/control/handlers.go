package control

import (
	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
)

// Domain teams add one handler set to this list. A set implements only its own
// generated strict methods, plus Replay for commands. Overlap fails at startup.
func Handlers(pool *pgxpool.Pool, logger *slog.Logger) (*httpapi.Composite, error) {
	return httpapi.Compose(
		memberHandlers(pool),
		operations.Store{Pool: pool},
		&audit.Store{Pool: pool, Logger: logger},
	)
}
