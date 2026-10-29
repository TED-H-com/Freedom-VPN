package control

import (
	"net/http"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

func memberHandlers(pool *pgxpool.Pool) httpadapter.Handlers { return httpadapter.Handlers{Pool: pool} }

// ConfigureMembers supplies central configuration and the original signature
// header to the strict handlers. Call after Servers and before Run. Construction
// never makes an external request; secrets remain central and errors are safe.
func ConfigureMembers(c config.Config, pool *pgxpool.Pool, public *http.Server) error {
	cfg, e := config.LoadMembers(c.MembersConfigFile)
	if e != nil {
		return e
	}
	services, e := membersruntime.New(pool, cfg, nil)
	if e != nil {
		return e
	}
	public.Handler = httpadapter.Transport(public.Handler, &services.HTTP)
	return nil
}
