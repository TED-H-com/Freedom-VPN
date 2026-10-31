package installer

import (
	"context"
	"encoding/json"
	"time"
)

const ownerComment = "managed-by=fvpn-node-install"

// MarkerStatus describes ownership only, never forwarding or dataplane health.
type MarkerStatus struct {
	Table       string `json:"table"`
	Present     bool   `json:"present"`
	LegacyOwner bool   `json:"legacy_owner"`
}

func readMarker(ctx context.Context, env *Env) (MarkerStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	raw, err := env.output(ctx, "nft", "-j", "list", "ruleset")
	if err != nil {
		return MarkerStatus{Table: env.layout().markerTable()}, errf(ExitFailed, "NFT_MARKER", "cannot read nft ownership")
	}
	return parseMarker(raw, env.layout())
}

func parseMarker(raw []byte, l Layout) (MarkerStatus, error) {
	st := MarkerStatus{Table: l.markerTable()}
	bad := func() (MarkerStatus, error) {
		return st, errf(ExitFailed, "NFT_MARKER", "unexpected installer marker; refusing to change nft objects")
	}
	var doc struct {
		Nftables []map[string]json.RawMessage `json:"nftables"`
	}
	if json.Unmarshal(raw, &doc) != nil || doc.Nftables == nil {
		return bad()
	}
	var tables, owners, legacy int
	for _, obj := range doc.Nftables {
		for kind, raw := range obj {
			var v struct {
				Family  string `json:"family"`
				Table   string `json:"table"`
				Name    string `json:"name"`
				Chain   string `json:"chain"`
				Comment string `json:"comment"`
			}
			if json.Unmarshal(raw, &v) != nil {
				return bad()
			}
			if v.Family != "inet" {
				continue
			}
			if kind == "table" && v.Name == st.Table {
				tables++
				continue
			}
			marker := v.Table == st.Table
			old := v.Table == l.NFTTable && (kind == "chain" && v.Name == "owner" || v.Chain == "owner")
			if !marker && !old {
				continue
			}
			// Only the exact regular empty chain is ours. Reject hooks, rules,
			// extra chains/sets/objects, or an operator's same-name chain.
			if kind != "chain" || v.Name != "owner" || v.Comment != ownerComment {
				return bad()
			}
			var fields map[string]json.RawMessage
			if json.Unmarshal(raw, &fields) != nil {
				return bad()
			}
			for key := range fields {
				switch key {
				case "family", "table", "name", "handle", "comment":
				default:
					return bad()
				}
			}
			if marker {
				owners++
			} else {
				legacy++
			}
		}
	}
	if tables > 1 || owners > 1 || legacy > 1 || tables != owners {
		return bad()
	}
	st.Present, st.LegacyOwner = owners == 1, legacy == 1
	return st, nil
}

// reconcileMarker also runs on the already_current and upgrade paths, under
// the shared update lock, so old installations cannot bypass migration.
func reconcileMarker(ctx context.Context, env *Env) error {
	if err := prepareJournal(env); err != nil {
		return err
	}
	j, err := openJournal(env)
	if err != nil {
		return err
	}
	if _, err := ensureTable(ctx, env, j); err != nil {
		_ = j.Rollback(ctx)
		return err
	}
	if err := writeOwned(env, j); err != nil {
		_ = j.Rollback(ctx)
		return err
	}
	return j.Finish()
}

func removeMarker(ctx context.Context, env *Env) error {
	st, err := readMarker(ctx, env)
	if err != nil {
		return err
	}
	if st.Present {
		if err := env.run(ctx, "nft", "delete", "table", "inet", st.Table); err != nil {
			return err
		}
		after, err := readMarker(ctx, env)
		if err != nil {
			return err
		}
		if after.Present {
			return errf(ExitFailed, "NFT_MARKER", "installer marker removal read-back mismatch")
		}
	}
	return nil
}
