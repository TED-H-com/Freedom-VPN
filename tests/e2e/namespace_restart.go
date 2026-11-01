//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

type RestartSample struct {
	At        time.Time `json:"at"`
	Heartbeat string    `json:"heartbeat"`
	Error     string    `json:"error,omitempty"`
}
type NamespaceRestartEvidence struct {
	NodeID           uuid.UUID       `json:"node_id"`
	Slot             string          `json:"slot"`
	Started          time.Time       `json:"started_at"`
	Finished         time.Time       `json:"finished_at"`
	GenerationBefore string          `json:"generation_before"`
	GenerationAfter  string          `json:"generation_after"`
	BootReadback     string          `json:"boot_readback"`
	EndReadback      string          `json:"end_readback"`
	Samples          []RestartSample `json:"samples"`
	Observations     json.RawMessage `json:"observations"`
	AgentLog         string          `json:"agent_log"`
	ServingAt        *time.Time      `json:"serving_at"`
}

// RestartNodeNamespace90 destroys and recreates only the selected owned lab
// namespace; durable identity, desired state and usage remain. It always samples
// the full 90 seconds, even when serving returns earlier. No generation bump,
// state clearing, interface repair or forced lease is performed.
func (s *Stack) RestartNodeNamespace90(ctx context.Context, slot string, node uuid.UUID) (out NamespaceRestartEvidence, err error) {
	out.Slot, out.NodeID = slot, node
	if slot != "node" && slot != "node2" {
		return out, fmt.Errorf("restart requires node or node2")
	}
	c, err := s.nodeSlot(slot)
	if err != nil {
		return out, err
	}
	db, err := s.lab.db(ctx)
	if err != nil {
		return out, err
	}
	defer db.Close(ctx)
	generation := func(dest *string) error {
		return db.QueryRow(ctx, `SELECT desired_config_generation::text FROM nodes WHERE id=$1`, node).Scan(dest)
	}
	if err = generation(&out.GenerationBefore); err != nil {
		return out, err
	}
	dns := c + "-adguard"
	if slot == "node" {
		dns = s.lab.prefix + "-adguard"
	}
	ns := c + "-netns"
	if err = exec.CommandContext(ctx, "docker", "stop", "--time", "25", c).Run(); err != nil {
		return out, err
	}
	if err = exec.CommandContext(ctx, "docker", "stop", "--time", "5", dns, ns).Run(); err != nil {
		return out, err
	}
	out.Started = time.Now().UTC()
	for _, name := range []string{ns, dns, c} {
		if err = exec.CommandContext(ctx, "docker", "start", name).Run(); err != nil {
			return out, err
		}
	}
	readback := func() string {
		b, e := s.lab.docker(c, "sh", "-c", `date -u +%FT%TZ; nft -j list table inet fvpn; for d in fvmb0 fvup0; do tc -j qdisc show dev "$d"; tc -j class show dev "$d"; tc -j filter show dev "$d"; done`)
		if e != nil {
			return string(b) + "\nreadback error: " + e.Error()
		}
		return string(b)
	}
	out.BootReadback = readback()
	deadline := out.Started.Add(90 * time.Second)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		beat, e := s.lab.docker(c, "cat", "/run/vpn-node/heartbeat")
		sample := RestartSample{At: time.Now().UTC(), Heartbeat: string(beat)}
		if e != nil {
			sample.Error = "heartbeat unavailable"
		}
		out.Samples = append(out.Samples, sample)
		if out.ServingAt == nil {
			var at time.Time
			e = db.QueryRow(ctx, `SELECT received_at FROM node_observations WHERE node_id=$1 AND received_at >= $2 AND report_payload->>'member_forwarding_enabled'='true' ORDER BY received_at LIMIT 1`, node, out.Started).Scan(&at)
			if e == nil {
				at = at.UTC()
				out.ServingAt = &at
			}
		}
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	out.Finished = time.Now().UTC()
	out.EndReadback = readback()
	log, _ := s.lab.docker(c, "cat", "/var/lib/vpn-node/e2e/agent.log")
	out.AgentLog = Redact(string(log))
	if err = generation(&out.GenerationAfter); err != nil {
		return out, err
	}
	err = db.QueryRow(ctx, `SELECT coalesce(jsonb_agg(jsonb_build_object('received_at',received_at,'report',report_payload) ORDER BY received_at),'[]'::jsonb) FROM node_observations WHERE node_id=$1 AND received_at >= $2 AND received_at <= $3`, node, out.Started, out.Finished).Scan(&out.Observations)
	return out, err
}
