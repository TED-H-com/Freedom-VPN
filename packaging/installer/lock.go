package installer

import (
	"context"
	"encoding/json"
	"os"

	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
)

var errBusy = nodelock.ErrBusy

// The shared lock uses the kernel boot id first. A configured boot_id is
// only a fallback; an absent updater config is normal during initial install.
func tryLock(ctx context.Context, l Layout) (*nodelock.Lock, error) {
	var cfg struct {
		BootID string `json:"boot_id"`
	}
	if raw, err := os.ReadFile(l.updaterConfig()); err == nil {
		_ = json.Unmarshal(raw, &cfg)
	}
	return nodelock.AcquireWithBootID(ctx, l.LockPath, 1, 0, cfg.BootID)
}
