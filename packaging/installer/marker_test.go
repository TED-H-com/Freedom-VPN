package installer

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

const markerFixture = `{"nftables":[{"table":{"family":"inet","name":"fvpn_install"}},{"chain":{"family":"inet","table":"fvpn_install","name":"owner","handle":1,"comment":"managed-by=fvpn-node-install"}}]}`
const legacyFixture = `{"chain":{"family":"inet","table":"fvpn","name":"owner","handle":2,"comment":"managed-by=fvpn-node-install"}}`

func TestPKG01_D116MarkerRefusesUnknownObjects(t *testing.T) {
	for name, raw := range map[string]string{
		"invalid JSON":        "not json",
		"missing listing":     `{}`,
		"foreign owner":       strings.Replace(markerFixture, ownerComment, "operator-owned", 1),
		"base chain":          strings.Replace(markerFixture, `"handle":1`, `"handle":1,"hook":"forward"`, 1),
		"nonempty":            strings.TrimSuffix(markerFixture, "]}") + `,{"rule":{"family":"inet","table":"fvpn_install","chain":"owner","expr":[{"accept":null}]}}]}`,
		"extra object":        strings.TrimSuffix(markerFixture, "]}") + `,{"set":{"family":"inet","table":"fvpn_install","name":"other"}}]}`,
		"empty unowned table": `{"nftables":[{"table":{"family":"inet","name":"fvpn_install"}}]}`,
		"legacy hook":         `{"nftables":[` + strings.Replace(legacyFixture, `"handle":2`, `"handle":2,"hook":"input"`, 1) + `]}`,
		"legacy rule":         `{"nftables":[` + legacyFixture + `,{"rule":{"family":"inet","table":"fvpn","chain":"owner"}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			env := baseEnv(t, t.TempDir())
			mutations := 0
			env.Command = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				if strings.Join(args, " ") == "-j list ruleset" {
					return []byte(raw), nil
				}
				mutations++
				return nil, nil
			}
			if err := reconcileMarker(context.Background(), env); err == nil {
				t.Fatal("accepted unowned marker")
			}
			if err := removeMarker(context.Background(), env); err == nil {
				t.Fatal("removed unowned marker")
			}
			if mutations != 0 {
				t.Fatalf("mutated unknown objects: %d", mutations)
			}
		})
	}
}

func TestAT07_D116MigrateOnRerunAndUpgrade(t *testing.T) {
	for _, upgrade := range []bool{false, true} {
		name := "already_current"
		if upgrade {
			name = "upgrade"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			env := baseEnv(t, root)
			fx := mustRelease(t, root, "019a0000-0000-7000-8000-0000000000c1", "1")
			req := mustReq(t, root, fx)
			ctx := context.Background()
			if _, err := Install(ctx, env, req); err != nil {
				t.Fatal(err)
			}
			// Emulate follow-up 3's persisted ownership record and empty chain.
			if err := os.WriteFile(env.layout().ownedFile(), []byte(`{"created_table":true}`), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := removeMarker(ctx, env); err != nil {
				t.Fatal(err)
			}
			legacy := true
			command := env.Command
			env.Command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				out, err := command(ctx, name, args...)
				if name == "nft" && strings.Join(args, " ") == "-j list ruleset" && legacy {
					out = []byte(strings.Replace(string(out), `"nftables":[`, `"nftables":[`+legacyFixture+`,`, 1))
					out = []byte(strings.Replace(string(out), ",]", "]", 1))
				}
				if name == "nft" && strings.Join(args, " ") == "delete chain inet fvpn owner" {
					legacy = false
				}
				return out, err
			}
			if upgrade {
				next := mustRelease(t, root, "019a0000-0000-7000-8000-0000000000c2", "2")
				if _, err := Upgrade(ctx, env, next.Manifest, next.Artifact, next.Trust); err != nil {
					t.Fatal(err)
				}
			} else {
				for i := 0; i < 2; i++ {
					res, err := Install(ctx, env, req)
					if err != nil || res.Status != statusAlreadyCurrent {
						t.Fatalf("rerun: %+v %v", res, err)
					}
				}
			}
			rep, err := Check(ctx, env)
			if err != nil || !rep.Marker.Present || rep.Marker.LegacyOwner || legacy {
				t.Fatalf("migration check: %+v %v", rep, err)
			}
			calls := joinedCalls(env)
			if strings.Count(calls, "nft delete chain inet fvpn owner") != 1 || strings.Count(calls, "nft create table inet fvpn_install") != 2 {
				t.Fatalf("non-idempotent migration:\n%s", calls)
			}
			for _, purge := range []bool{false, true} {
				if err := reconcileMarker(ctx, env); err != nil {
					t.Fatal(err)
				}
				if _, err := Remove(ctx, env, purge); err != nil {
					t.Fatal(err)
				}
				st, err := readMarker(ctx, env)
				if err != nil || st.Present {
					t.Fatalf("remove purge=%t: %+v %v", purge, st, err)
				}
			}
			for _, line := range strings.Split(joinedCalls(env), "\n") {
				if strings.HasPrefix(line, "nft ") && (strings.Contains(line, "flush") || line == "nft delete table inet fvpn" || strings.Contains(line, "add chain inet fvpn ")) {
					t.Fatalf("changed agent/foreign rules: %s", line)
				}
			}
		})
	}
}

func TestPKG01_D116ReadFailureAndLegacyRollback(t *testing.T) {
	env := baseEnv(t, t.TempDir())
	command := env.Command
	env.Command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "nft" {
			return nil, errors.New("permission denied")
		}
		return command(ctx, name, args...)
	}
	if err := reconcileMarker(context.Background(), env); err == nil {
		t.Fatal("unreadable ruleset accepted")
	}
	j := &Journal{env: env}
	if err := j.undo(context.Background(), Record{Action: "nft-table", Data: "fvpn"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(joinedCalls(env), "delete table") {
		t.Fatal("old journal deleted agent table")
	}
}
