// Package runtime constructs the member area's explicitly configured dependencies.
package runtime

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Compiler interface {
	RecompileBatch(context.Context, []db.ClaimedEvent) ([]desiredstate.CompiledDesiredState, error)
}
type Services struct {
	HTTP                              httpadapter.Services
	Adapter                           *devices.NetBirdAdapter
	Compiler                          Compiler
	ReconcileInterval, DeviceInterval time.Duration
}

func New(pool *pgxpool.Pool, c config.Members, compiler Compiler) (*Services, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	if pool == nil {
		return nil, errors.New("member database required")
	}
	ec, e := c.Entitlements()
	if e != nil {
		return nil, e
	}
	engine, e := entitlement.NewStore(ec)
	if e != nil {
		return nil, e
	}
	provider, e := billing.NewStripe(billing.StripeConfig{APIKeyFile: c.APIKeyFile, WebhookSecretFiles: c.WebhookSecretFiles, LiveMode: *c.LiveMode, PortalSessionTTL: time.Duration(c.PortalTTLSeconds) * time.Second})
	if e != nil {
		return nil, e
	}
	keys := map[string][]byte{}
	defer func() {
		for _, key := range keys {
			clear(key)
		}
	}()
	for id, path := range c.VaultKeyFiles {
		key, e := os.ReadFile(path)
		if e != nil {
			return nil, errors.New("cannot read billing vault key")
		}
		keys[id] = key
	}
	vault, e := billing.NewVault(c.VaultActiveKey, keys)
	if e != nil {
		return nil, e
	}
	bs, e := billing.NewService(pool, provider, vault, engine, billing.Config{FullRefundPolicy: c.FullRefundPolicy, ReturnURLs: c.ReturnURLs, LeaseDuration: time.Duration(c.BillingLeaseSeconds) * time.Second, MaxWebhookBytes: c.MaxWebhookBytes, AllowedPriceIDs: c.PriceIDs})
	if e != nil {
		return nil, e
	}
	bindings := devices.NetBirdBindings{Pool: pool}
	nb, e := netbird.New(netbird.Config{BaseURL: c.NetBirdURL, DeploymentID: c.NetBirdDeployment, Token: netbird.TokenFile(c.NetBirdTokenFile), Bindings: bindings})
	if e != nil {
		return nil, e
	}
	adapter := &devices.NetBirdAdapter{Pool: pool, Client: nb, Bindings: bindings, DeploymentID: c.NetBirdDeployment, OverlayScope: c.OverlayScope}
	return &Services{HTTP: httpadapter.Services{Members: &members.Service{Pool: pool, Entitlements: engine}, Devices: &devices.Service{Pool: pool, Entitlements: engine, Adapter: adapter}, Billing: bs}, Adapter: adapter, Compiler: compiler, ReconcileInterval: time.Duration(c.ReconcileSeconds) * time.Second, DeviceInterval: time.Duration(c.DeviceSyncSeconds) * time.Second}, nil
}
