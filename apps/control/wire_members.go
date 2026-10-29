package control

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
)

func members() Domain {
	return Domain{Name: "members", Build: func(_ context.Context, d Dependencies) (Built, error) {
		cfg, e := config.LoadMembers(d.Config.MembersConfigFile)
		if e != nil {
			return Built{}, &config.SettingError{Name: "members_config_file", Err: e}
		}
		services, e := membersruntime.New(d.Pool, cfg, nil)
		if e != nil {
			return Built{}, &config.SettingError{Name: "members_config_file", Err: e}
		}
		return Built{Handlers: []any{httpadapter.Handlers{Pool: d.Pool, Services: &services.HTTP}}, Middleware: httpadapter.RequestContext}, nil
	}}
}
