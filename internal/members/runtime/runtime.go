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

func New(pool *pgxpool.Pool, c config.Members, identity *devices.IdentityDiscovery, compiler Compiler) (*Services, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	if identity == nil {
		return nil, errors.New("member identity discovery required")
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
	// Validate the configured file at startup while retaining TokenFile's
	// per-request reload for rotation. Never retain or log the returned token.
	token := netbird.TokenFile(c.NetBirdTokenFile)
	if _, e := token.Token(context.Background()); e != nil {
		return nil, errors.New("cannot read NetBird token")
	}
	bindings := devices.NetBirdBindings{Pool: pool}
	nb, e := netbird.New(netbird.Config{BaseURL: c.NetBirdURL, DeploymentID: c.NetBirdDeployment, Token: token, Bindings: bindings})
	if e != nil {
		return nil, e
	}
	adapter := &devices.NetBirdAdapter{Pool: pool, Client: nb, Bindings: bindings, DeploymentID: c.NetBirdDeployment, OverlayScope: c.OverlayScope, Identity: identity}
	return &Services{HTTP: httpadapter.Services{Members: &members.Service{Pool: pool, Entitlements: engine}, Devices: &devices.Service{Pool: pool, Entitlements: engine, Adapter: adapter}, Billing: bs}, Adapter: adapter, Compiler: compiler, ReconcileInterval: time.Duration(c.ReconcileSeconds) * time.Second, DeviceInterval: time.Duration(c.DeviceSyncSeconds) * time.Second}, nil
}

// WithCompiler lets the policy/node domain supply its configured production
// compiler via the worker composition context, without ordering domain builds or
// inserting per-domain startup calls. No default rates or signer are invented.
type compilerContextKey struct{}

func WithCompiler(ctx context.Context, compiler Compiler) context.Context {
	return context.WithValue(ctx, compilerContextKey{}, compiler)
}
func CompilerFromContext(ctx context.Context) Compiler {
	compiler, _ := ctx.Value(compilerContextKey{}).(Compiler)
	return compiler
}
