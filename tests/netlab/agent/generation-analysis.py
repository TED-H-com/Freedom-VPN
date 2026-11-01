"""TEST ONLY: summarize original F31 producer/packet records, never waive losses."""
import argparse
import json
import math
from pathlib import Path


def distribution(values):
    values = sorted(values)
    if not values:
        return {"count": 0, "available": False}
    return dict(count=len(values), available=True,
                **{key: str(values[max(0, math.ceil(len(values) * q) - 1)])
                   for key, q in (("p50", .5), ("p90", .9), ("p99", .99), ("max", 1))},
                values=[str(x) for x in values])


def analyze(root):
    rows = [json.loads(x) for x in (root / "operations.jsonl").read_text().splitlines()]
    transitions = json.loads((root / "transitions.json").read_text())
    capture = json.loads((root / "continuous.json").read_text())
    packets = capture["packets"]
    prior = {}
    for row in rows:
        key = (row["run_id"], row["boot_id"])
        assert int(row["sequence"]) == prior.get(key, 0) + 1, "producer sequence gap"
        prior[key] = int(row["sequence"])
    assert len(prior) == 1, "generation experiment requires one producer/boot"
    assert len(transitions) >= 40, "incomplete experiment"
    assert all(s["drops"] == 0 for s in capture["capture_stats"]), "capture loss"
    assert all(int(p["ingress_ns"]) > 0 for p in packets), "missing ingress"
    applies = [r for r in rows if r.get("timing", {}).get("operation") == "apply"]
    guards = [(int(r["timing"]["deadline"]["install_start_ns"]),
               int(r["timing"]["deadline"]["release_end_ns"]))
              for r in applies if r["timing"]["deadline"]["guard_released"]]
    def in_guard(p):
        return any(lo <= int(p["ingress_ns"]) < hi for lo, hi in guards)
    sibling = [p for p in packets if p["source"] == "100.64.1.2"]
    losses = [p for p in sibling if int(p["egress_ns"]) == 0]
    for source in ("100.64.1.1", "100.64.1.2"):
        assert any(p["source"] == source and 0 < int(p["egress_ns"]) < int(transitions[0]["start_ns"])
                   for p in packets), "missing initial two-source positive"
    results = []
    vectors = {}
    for index, transition in enumerate(transitions):
        selected = [r for r in rows if int(transition["sequence_before"]) < int(r["sequence"]) <= int(transition["sequence_after"])]
        baseline = [r for r in selected if r.get("baseline", {}).get("cause") == "policy_missing"]
        renewal = [r for r in selected if r.get("renewal", {}).get("new_lease")]
        apply = [r for r in selected if r.get("timing", {}).get("operation") == "apply"]
        assert baseline and all(r["baseline"]["succeeded"] for r in baseline), "missing successful baseline"
        assert all(r["lease"]["generation"] == transition["old_lease_generation"] for r in baseline)
        assert len(renewal) == len(apply) == 1, "unexpected renewal/apply count"
        renewal, apply = renewal[0], apply[0]
        a, d = apply["timing"], apply["timing"]["deadline"]
        assert a["succeeded"] and a["verified"] and d["guard_released"], "failed candidate"
        assert renewal["lease"]["generation"] == a["candidate_generation"] == transition["generation"]
        assert all(v["verified"] for v in a["verifications"])
        live = [r for r in selected if r["source"] == "live" and int(r["timing"]["start_ns"]) >= int(a["end_ns"])]
        beats = [r for r in selected if r.get("heartbeat", {}).get("serving") and r["heartbeat"]["generation"] == transition["generation"]]
        assert live and beats and all(r["timing"]["verified"] for r in live), "recovery missing verification/heartbeat"
        begin = int(baseline[0]["baseline"]["start_ns"])
        end = int(a["end_ns"])
        captured_losses = [p for p in losses if begin <= int(p["ingress_ns"]) < end]
        step_losses = [p for p in losses if int(transition["start_ns"]) <= int(p["ingress_ns"]) < int(transition["end_ns"])]
        next_start = int(transitions[index+1]["start_ns"]) if index+1 < len(transitions) else int(capture["end_ns"])
        after = [p for p in sibling if end <= int(p["ingress_ns"]) < next_start and int(p["egress_ns"]) > 0]
        assert after, "no independent sibling recovery"
        quiet = [p for p in packets if int(transition["end_ns"]) <= int(p["ingress_ns"]) < next_start]
        assert any(p["source"] == "100.64.1.2" and int(p["egress_ns"]) for p in quiet), "no between-transition positive"
        if not transition["first_member_admitted"]:
            assert not any(p["source"] == "100.64.1.1" and int(p["egress_ns"]) for p in quiet), "revoked source forwarded after recovery"
        metrics = {
            "baseline_to_apply_end_ns": end-begin,
            "baseline_to_serving_heartbeat_ns": int(beats[0]["heartbeat"]["mono_ns"])-begin,
            "baseline_to_first_observed_post_apply_ingress_ns": int(after[0]["ingress_ns"])-begin,
            "first_baseline_command_ns": int(baseline[0]["baseline"]["end_ns"])-begin,
            "renewal_ns": int(renewal["renewal"]["end_ns"])-int(renewal["renewal"]["start_ns"]),
            "apply_ns": end-int(a["start_ns"]),
            "apply_verification_ns": sum(int(v["end_ns"])-int(v["start_ns"]) for v in a["verifications"]),
            "post_apply_live_verification_ns": int(live[0]["timing"]["end_ns"])-int(live[0]["timing"]["start_ns"]),
            "outer_guard_ns": int(d["release_end_ns"])-int(d["install_start_ns"]),
            "render_to_installed_ns": int(d["installed_by_ns"])-int(d["render_start_ns"]),
            "baseline_to_apply_losses": len(captured_losses),
            "baseline_to_apply_losses_inside_guards": sum(in_guard(p) for p in captured_losses),
            "baseline_to_apply_losses_outside_guards": sum(not in_guard(p) for p in captured_losses),
        }
        for key, value in metrics.items():
            vectors.setdefault(key, []).append(value)
        results.append(dict(transition=transition, baseline_sequences=[r["sequence"] for r in baseline],
                            renewal_sequence=renewal["sequence"], apply_sequence=apply["sequence"],
                            live_sequences=[r["sequence"] for r in live], serving_sequence=beats[0]["sequence"],
                            metrics={k: str(v) for k, v in metrics.items()},
                            losses=captured_losses, whole_step_losses=step_losses,
                            first_post_apply_packet=after[0]))
    commands = [c for r in applies for c in r["timing"].get("commands", [])]
    assert commands, "lifecycle observer not enabled"
    for c in commands:
        assert not c["child_start_exact"]["available"], "invented exact child start"
        if c["start_attempt"]["available"] and c["start_return"]["available"]:
            assert int(c["start_attempt"]["ns"]) <= int(c["start_return"]["ns"])
        bracket = c["context_deadline"]
        if bracket["before"]["available"] and bracket["after"]["available"]:
            assert int(bracket["before"]["ns"]) <= int(bracket["after"]["ns"])
    return dict(scope="TEST ONLY local TLS control/real kernel; original records retained; no availability SLA",
                percentile_method="nearest rank; nanoseconds and loss counts as decimal strings",
                run_id=rows[0]["run_id"], boot_id=rows[0]["boot_id"], transitions=results,
                distributions={k: distribution(v) for k, v in vectors.items()},
                packet_totals=dict(sibling_ingress=len(sibling), sibling_egress=len(sibling)-len(losses),
                                   sibling_losses=len(losses), inside_completed_guards=sum(in_guard(p) for p in losses),
                                   outside_completed_guards=sum(not in_guard(p) for p in losses)),
                all_sibling_losses=losses, capture_stats=capture["capture_stats"],
                failed_applies=[r for r in applies if not r["timing"]["succeeded"]],
                lifecycle=dict(commands=len(commands), started=sum(c["started"] for c in commands),
                               cpu_available=sum(c["cpu_available"] for c in commands),
                               switches_available=sum(c["switches_available"] for c in commands),
                               exact_child_start_available=sum(c["child_start_exact"]["available"] for c in commands),
                               hidden_host_ancestors_available=sum(c["before"]["host_ancestors_available"] for c in commands),
                               cause_status="unproven; observations do not establish contention causality"))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    args.output.write_text(json.dumps(analyze(args.directory), indent=2)+"\n")
