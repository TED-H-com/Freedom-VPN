package control

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
)

func members() Domain {
	return Domain{Name: "members", Build: func(_ context.Context, d Dependencies) (Built, error) {
		cfg, e := config.LoadMembersForIssuer(d.Config.MembersConfigFile, d.Config.OIDCIssuer)
		if e != nil {
			return Built{}, e
		}
		identity, e := devices.NewIdentityDiscovery(d.Config.OIDCIssuer, nil, d.Logger)
		if e != nil {
			return Built{}, &config.SettingError{Name: "oidc_issuer", Err: e}
		}
		services, e := membersruntime.New(d.Pool, cfg, identity, nil)
		if e != nil {
			return Built{}, &config.SettingError{Name: "members_config_file", Err: e}
		}
		// Handlers owns listAdminDevices, authorizeMemberDevice and revokeMemberGrant.
		// Composition binds the methods; there is no separate route table.
		return Built{Handlers: []any{httpadapter.Handlers{Pool: d.Pool, Services: &services.HTTP}}, Middleware: httpadapter.RequestContext}, nil
	}}
}
