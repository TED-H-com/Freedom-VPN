package installer

import (
	"context"
	"encoding/json"
	"os"
	"strings"
)

// initIsolation creates the installer marker and an empty netns when they are
// absent (§6.3, AT13). It never flushes the host ruleset, never deletes a
// table it did not create, and never moves a host interface. enforce
// programs inet fvpn in the host network namespace, so the data-plane
// interfaces stay there. The netns name is the namespace this installer owns.
func initIsolation(ctx context.Context, env *Env, j *Journal, member, uplink string) error {
	if _, err := ensureTable(ctx, env, j); err != nil {
		return err
	}
	if err := ensureNetns(ctx, env, j); err != nil {
		return err
	}
	if err := ensureLink(ctx, env, j, member); err != nil {
		return err
	}
	if err := ensureLink(ctx, env, j, uplink); err != nil {
		return err
	}
	// systemd refuses ReadWritePaths that do not exist (226/NAMESPACE).
	// Create the NetBird directories before any unit starts (§6.3).
	if err := j.MkdirAll(env.layout().NetbirdEtc, 0o755); err != nil {
		return err
	}
	if err := j.MkdirAll(env.layout().NetbirdLib, 0o700); err != nil {
		return err
	}
	return writeOwned(env, j)
}

// ownedState is the set of network objects this installer created.
// remove deletes only this set (§6.3).
type ownedState struct {
	CreatedMarkerTable bool     `json:"created_marker_table"`
	CreatedNetns       bool     `json:"created_netns"`
	CreatedLinks       []string `json:"created_links"`
	InstalledNetbird   bool     `json:"installed_netbird"`
}

func readOwned(l Layout) ownedState {
	raw, err := os.ReadFile(l.ownedFile())
	if err != nil {
		return ownedState{}
	}
	var st ownedState
	if json.Unmarshal(raw, &st) != nil {
		return ownedState{}
	}
	return st
}

func writeOwned(env *Env, j *Journal) error {
	prev := readOwned(env.layout())
	st := prev
	for _, rec := range j.file.Records {
		switch rec.Action {
		case "nft-marker-table":
			st.CreatedMarkerTable = true
		case "netns":
			st.CreatedNetns = true
		case "link":
			if !contains(st.CreatedLinks, rec.Data) {
				st.CreatedLinks = append(st.CreatedLinks, rec.Data)
			}
		case "package":
			if rec.Data == "netbird" {
				st.InstalledNetbird = true
			}
		}
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return j.WriteFile(env.layout().ownedFile(), raw, 0o600)
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func ensureTable(ctx context.Context, env *Env, j *Journal) (bool, error) {
	st, err := readMarker(ctx, env)
	if err != nil {
		return false, err
	}
	table := env.layout().markerTable()
	if !st.Present {
		if err := j.Note("nft-marker-table", "inet "+table, table, "created hookless installer marker"); err != nil {
			return false, err
		}
		// One atomic transaction, with exclusive creation. No partial empty
		// table and no objects added to the agent's inet fvpn (D-116).
		if err := env.run(ctx, "nft", "create", "table", "inet", table,
			`{ chain owner { comment "managed-by=fvpn-node-install"; } }`); err != nil {
			return false, err
		}
	}
	if st.LegacyOwner {
		// Forward-only migration: rollback must never resurrect the fifth
		// chain that prevents serving. Delete only the exact empty marker.
		if err := j.Note("nft-owner-migration", "inet "+env.layout().NFTTable, "owner", "removed legacy empty installer marker"); err != nil {
			return false, err
		}
		if err := env.run(ctx, "nft", "delete", "chain", "inet", env.layout().NFTTable, "owner"); err != nil {
			return false, err
		}
	}
	return !st.Present || st.LegacyOwner, nil
}

func ensureNetns(ctx context.Context, env *Env, j *Journal) error {
	name := env.layout().NetnsName
	out, err := env.output(ctx, "ip", "netns", "list")
	if err == nil && netnsPresent(string(out), name) {
		return nil
	}
	if err := j.Note("netns", name, name, "created netns"); err != nil {
		return err
	}
	return env.run(ctx, "ip", "netns", "add", name)
}

func netnsPresent(list, name string) bool {
	for _, line := range strings.Split(list, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == name {
			return true
		}
	}
	return false
}

func ensureLink(ctx context.Context, env *Env, j *Journal, name string) error {
	if _, err := env.output(ctx, "ip", "link", "show", name); err == nil {
		return nil
	}
	if err := j.Note("link", name, name, "created dummy"); err != nil {
		return err
	}
	// Dummy only. No address and no route (§6.3).
	return env.run(ctx, "ip", "link", "add", name, "type", "dummy")
}

// removeOwnedNetwork deletes only the objects owned.json lists.
func removeOwnedNetwork(ctx context.Context, env *Env) error {
	st := readOwned(env.layout())
	if err := removeMarker(ctx, env); err != nil {
		return err
	}
	for i := len(st.CreatedLinks) - 1; i >= 0; i-- {
		_ = env.run(ctx, "ip", "link", "delete", st.CreatedLinks[i])
	}
	if st.CreatedNetns {
		_ = env.run(ctx, "ip", "netns", "delete", env.layout().NetnsName)
	}
	return nil
}
