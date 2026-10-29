package control

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
)

func nodes() Domain {
	return Domain{Name: "nodes", Build: func(ctx context.Context, d Dependencies) (Built, error) {
		cfg, err := config.LoadNodes(d.Config.NodesConfigFile)
		if err != nil {
			return Built{}, &config.SettingError{Name: "nodes_config_file", Err: err}
		}
		runtime, err := httpwire.Open(ctx, d.Pool, cfg, d.Logger)
		if err != nil {
			return Built{}, &config.SettingError{Name: "nodes_config_file", Err: err}
		}
		return Built{Handlers: []any{&httpwire.Handlers{Runtime: runtime}}, Agents: runtime.PKI, Middleware: httpwire.RequestContext}, nil
	}}
}
