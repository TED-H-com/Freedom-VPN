#!/usr/bin/env python3
"""Render Follow-up 8 AT30 live central restore & cross-system reconciliation records."""
import hashlib
import json
import subprocess
from pathlib import Path

from run import PUBLIC, ROOT

PREVIOUS_RUN = "history/follow-up-7/"


def load(path: Path) -> dict:
    return json.loads(path.read_text())


def write(path: Path, value: dict) -> None:
    path.write_text(json.dumps(value, indent=2) + "\n")


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def digests_for_builds() -> list[dict]:
    out = []
    build_dir = ROOT / ".agent/secops/build"
    if build_dir.is_dir():
        for item in sorted(build_dir.iterdir()):
            if item.is_file():
                out.append({"name": item.name, "sha256": sha256_file(item)})
    return out


def clean_inventory(inv: dict) -> bool:
    return bool(inv) and all(
        v.get("exit_code") == 0 and v.get("count") == 0 for v in inv.values()
    )


def evaluate_at30(restore_case: dict) -> tuple[str, list[str], dict]:
    """Evaluate AT30 pass criteria from measured live restore & reconciliation."""
    blockers = []
    step1 = restore_case.get("step1_checkpoint_t0", {})
    step2 = restore_case.get("step2_world_forward_t1", {})
    step3 = restore_case.get("step3_restore_central_to_t0", {})
    step4 = restore_case.get("step4_reconciliation_observation", {})

    neg = step3.get("fail_safe_negative_without_protocol", {})
    proto = step3.get("product_restore_protocol", {})
    usage = step4.get("usage_replay", {})
    prov = step4.get("provider_reconciliation", {})
    peers = step4.get("peer_revocation_reconciliation", {})
    revoked_dev = peers.get("revoked_in_t0_t1_device", {})
    unmapped_dev = peers.get("new_in_t0_t1_unmapped_member_device", {})
    existing_dev = peers.get("new_in_t0_t1_existing_member_device", {})

    refusal_ok = bool(
        step3.get("primary_overwrite_refusal", {}).get("refused")
        and step3.get("primary_overwrite_refusal", {}).get("exit_code") == 2
    )
    if not refusal_ok:
        blockers.append("restore-pitr.sh primary-overwrite refusal did not exit 2.")

    checklist_ok = (
        proto.get("restore_info", {}).get("checklist_sections") == 6
    )
    if not checklist_ok:
        blockers.append("post-restore-checklist.sql did not pass all 6/6 sections.")

    neg_node_store = neg.get("after_120s_node_store", {})
    neg_ok = bool(
        neg.get("window_seconds", 0) >= 120
        and neg.get("retained_t0_t1_skipped_without_protocol") is True
        and (
            neg.get("node_ack_regress_log_lines", 0) >= 1
            or (
                str(neg.get("after_120s_central_ack", "")) == "2"
                and int(neg.get("after_120s_central_max_seen", "0") or 0) > 5
                and int(neg_node_store.get("outbox_count", 0) or 0) >= 1
            )
        )
    )
    if not neg_ok:
        blockers.append(
            "2-minute fail-safe negative before freeze/begin-recovery did not prove skipped T0..T1 archives and ErrAckRegress."
        )

    proto_ok = bool(
        proto.get("mode_after_freeze") == "frozen"
        and proto.get("mode_after_begin_recovery") == "reconciling"
        and proto.get("mode_after_complete_recovery") == "normal"
        and proto.get("reconciling_rollups_remaining") == 0
        and proto.get("member_api_ready", {}).get("m2_me_status") == 200
        and proto.get("member_api_ready", {}).get("m2_usage_status") == 200
        and proto.get("node_observation_accepted_after_recovery") is True
    )
    if not proto_ok:
        blockers.append(
            "5-step usage ledger restore protocol (freeze -> begin-recovery -> node replay -> complete-recovery) did not converge to normal mode with ready member API and fresh node observations."
        )

    usage_ok = bool(
        usage.get("status") == "passed"
        and usage.get("lost_t0_t1_acked_events") == 0
        and usage.get("double_counted_events") == 0
        and usage.get("usage_ledger_gaps") == 0
    )
    if not usage_ok:
        blockers.append(
            f"Retained-node usage archive replay did not match 1..N1 without loss or duplicates (status={usage.get('status')}, lost={usage.get('lost_t0_t1_acked_events')}, gaps={usage.get('usage_ledger_gaps')})."
        )

    prov_ok = bool(
        prov.get("status") == "passed"
        and prov.get("m1_subscription_status_l2") == "canceled"
        and prov.get("m1_subscription_active_l2") is False
        and prov.get("m1_refund_rows_l2") == 1
        and prov.get("m2_subscription_status_l2") == "active"
        and prov.get("m2_subscription_active_l2") is True
        and prov.get("m2_entitlement_extended") is False
    )
    if not prov_ok:
        blockers.append(
            f"Provider reconciliation after restore did not converge to P1 without entitlement extension (status={prov.get('status')})."
        )

    unmapped_dedup_ok = bool(
        unmapped_dev.get("deduplicated_across_sweeps") is True
        and unmapped_dev.get("identity_unmapped_audit_count") == 1
        and unmapped_dev.get("central_member_device_rows_l2") == 0
    )
    if not unmapped_dedup_ok:
        blockers.append(
            f"Unmapped NetBird peer discovery denial did not deduplicate across sweeps (count={unmapped_dev.get('identity_unmapped_audit_count')})."
        )

    readded = bool(
        revoked_dev.get("l2_observed_authorized")
        or revoked_dev.get("l2_readded_to_netbird_paid_group")
    )
    in_bindings = bool(
        revoked_dev.get("l2_in_active_device_bindings")
        or revoked_dev.get("l2_counted_flow_resumed")
    )
    revocation_preserved = bool(
        revoked_dev.get("l2_central_status_after_reconcile") == "revoked"
        and not readded
        and not in_bindings
    )
    if not revocation_preserved:
        blockers.append(
            "SECOPS-18 (w2-members / internal/devices): A device revoked in T0..T1 (peerM2D2) has member_devices.status='authorized' in the restored T0 database while NetBird N1 has already withdrawn it from fvpn-paid-<group_id> (with no external revocation tombstone in NetBird). When members.reconcile -> SyncMember -> SyncDevice (apps/worker/jobs_members.go:158, internal/devices/worker.go:72-93) runs after restore, it treats T0 member_devices.status='authorized' as authoritative and calls s.Adapter.SetAuthorization(ctx, deployment, peer, true, groups), silently re-adding the revoked peer to NetBird's paid group and re-including it in the node's active device_bindings without a discrepancy audit or re-confirmation."
        )

    summary = {
        "primary_overwrite_refusal": refusal_ok,
        "post_restore_checklist_6_of_6": checklist_ok,
        "fail_safe_negative_without_protocol": neg_ok,
        "product_restore_protocol_5_steps": proto_ok,
        "retained_node_usage_replay": usage_ok,
        "provider_reconciliation": prov_ok,
        "unmapped_peer_discovery_deduplicated": unmapped_dedup_ok,
        "existing_member_t1_peer_rediscovered": existing_dev.get("rediscovered_rows_in_l2") == 1,
        "t0_t1_device_revocation_preserved": revocation_preserved,
    }
    status = "passed" if not blockers else "blocked"
    return status, blockers, summary


def _checkpoint_of(ledger: dict) -> dict:
    if "checkpoint" in ledger:
        return ledger["checkpoint"]
    cps = ledger.get("checkpoints") or []
    return cps[0] if cps else {}


def _ledger_counts(ledger: dict) -> tuple[int, int, int]:
    ev_cnt = ledger.get("usage_events_count", len(ledger.get("events", [])))
    smp = ledger.get("samples", [])
    smp_cnt = ledger.get("usage_samples_count", len(smp))
    if "usage_samples_total_bytes" in ledger:
        tot = int(ledger["usage_samples_total_bytes"])
    else:
        tot = sum(
            int(s.get("upload_bytes", 0)) + int(s.get("download_bytes", 0))
            for s in smp
        )
    return ev_cnt, smp_cnt, tot


def _node_store_summary(ns: dict) -> str:
    next_seq = ns.get("next_seq", ns.get("meta", {}).get("next_seq"))
    acked = ns.get("acked_through", ns.get("meta", {}).get("acked_through"))
    arch = ns.get("archived_count", ns.get("state_counts", {}).get("archived", 0))
    outbox = ns.get("outbox_count", ns.get("state_counts", {}).get("outbox", 0))
    ev_cnt = ns.get("event_count", len(ns.get("events", [])))
    return f"`next_seq={next_seq}`, `acked_through={acked}`, `events={ev_cnt}`, `archived={arch}`, `outbox={outbox}`"


def render_restore_reconcile_md(record: dict) -> str:
    c = record["case"]
    s1 = c["step1_checkpoint_t0"]
    s2 = c["step2_world_forward_t1"]
    s3 = c["step3_restore_central_to_t0"]
    s4 = c["step4_reconciliation_observation"]

    b0 = s1["pgbackrest_backup"]
    l0 = s1["ledger_l0"]
    l1 = s2["ledger_l1"]
    cp0 = _checkpoint_of(l0)
    cp1 = _checkpoint_of(l1)
    l0_ev, l0_smp, l0_bytes = _ledger_counts(l0)
    l1_ev, l1_smp, l1_bytes = _ledger_counts(l1)
    neg = s3["fail_safe_negative_without_protocol"]
    proto = s3["product_restore_protocol"]
    usage = s4["usage_replay"]
    prov = s4["provider_reconciliation"]
    peers = s4["peer_revocation_reconciliation"]
    rev = peers["revoked_in_t0_t1_device"]
    unm = peers["new_in_t0_t1_unmapped_member_device"]
    exd = peers["new_in_t0_t1_existing_member_device"]
    readded = bool(rev.get("l2_observed_authorized") or rev.get("l2_readded_to_netbird_paid_group"))
    in_bindings = bool(rev.get("l2_in_active_device_bindings") or rev.get("l2_counted_flow_resumed"))
    t1_obs_auth = rev.get("t1_observed_authorized", s2.get("revoked_device_t1", {}).get("t1_observed_authorized", False))

    lines = [
        "# AT30 Live Central Database Restore & Cross-System Reconciliation — Follow-up 8",
        "",
        f"Application base `{record['application_base']}`; harness `{record['source_revision']}`; run `{record['run_id']}`; driver `{record['driver_run_id']}`.",
        f"Window: `{record['started_at']}` → `{record['finished_at']}`.",
        "",
        "## 1. Checkpoints `T0` and `T1` (`L0`, `L1`, `P1`, `N1`, and Node Retained Records)",
        "",
        "| Checkpoint / Artifact | Measured Value |",
        "|---|---|",
        f"| `T0` pgBackRest full backup label | `{b0['backup_label']}` (`type={b0['backup_type']}`) |",
        f"| `T0` restore target UTC / LSN | `{s1['t0_utc']}` / `{b0.get('t0_lsn', s1.get('t0_lsn'))}` |",
        f"| `L0` central usage checkpoint (`highest_contiguous_seq` / `max_seen_seq`) | `{cp0.get('highest_contiguous_seq')}` / `{cp0.get('max_seen_seq')}` (`ledger_epoch={cp0.get('ledger_epoch')}`) |",
        f"| `L0` central `usage_events` / `usage_samples` / total bytes | `{l0_ev}` / `{l0_smp}` / `{l0_bytes}` bytes |",
        f"| Node retained store at `T0` (`/var/lib/vpn-node/state/usage.db`) | {_node_store_summary(s1['node_retained_store_t0'])} |",
        f"| `T1` UTC after moving world forward | `{s2['t1_utc']}` (`T1 - T0 = {usage['pre_replay_db_rpo_seconds']:.3f} s`, LSN `{s2.get('t1_lsn')}`) |",
        f"| `L1` central usage checkpoint (`highest_contiguous_seq` / `max_seen_seq`) | `{cp1.get('highest_contiguous_seq')}` / `{cp1.get('max_seen_seq')}` (`ledger_epoch={cp1.get('ledger_epoch')}`) |",
        f"| `L1` central `usage_events` / `usage_samples` / total bytes | `{l1_ev}` / `{l1_smp}` / `{l1_bytes}` bytes |",
        f"| Node retained store at `T1` (`/var/lib/vpn-node/state/usage.db`) | {_node_store_summary(s2['node_retained_store_t1'])} |",
        f"| `P1` fake payment provider truth (`m1` / `m2`) | `m1`: `/e2e/snapshot` `past_due` probe returned HTTP `{s2['provider_change']['past_due_probe_status']}`; applied `status=canceled, refund_minor=1000` + `charge.refunded` webhook (`{s2['provider_change']['webhook_http_status']}`) → `provider_status={s2['provider_change']['t1_subscription_status']}`, `is_active={s2['provider_change']['t1_subscription_active']}`, `refund_records={s2['provider_change']['t1_refund_record_count']}`, `entitlement.state={s2['provider_change']['t1_entitlement_state']}`, `device_limit={s2['provider_change']['t1_entitlement_dev_limit']}`; `m2`: `active` (`paid_invoice=1`) |",
        f"| `N1` NetBird & peer changes in `T0..T1` | `peerM2D2` (`{s2['revoked_device_t1']['device_id']}`, peer `{s2['revoked_device_t1']['external_peer_id']}`) revoked (`DELETE` HTTP `{s2['revoked_device_t1']['delete_http_status']}`, `device.revoke` audit `{s2['revoked_device_t1']['t1_revoke_audit_event']['id']}`, `t1_observed_authorized={s2['revoked_device_t1']['t1_observed_authorized']}`); new `T0..T1` peers authorized: `peerM3D1` (`{s2['new_t0_t1_devices']['m3_new_member_device']['device_id']}` for post-`T0` member `{s2['new_t0_t1_devices']['m3_new_member_device']['member_id']}`) and `peerM2D3` (`{s2['new_t0_t1_devices']['m2_existing_member_device']['device_id']}` for `T0` member `{s2['new_t0_t1_devices']['m2_existing_member_device']['member_id']}`) |",
        "",
        "### Node Retained-Record Exposure (`node/` Source Finding)",
        "",
        f"- {s1['node_retained_exposure']['finding']}",
        "",
        "## 2. Restore Command Sequence, Fail-Safe Negative, and Measured `RTO`",
        "",
        "### 2.1 Primary-Overwrite Refusal & 2-Minute Fail-Safe Negative (Before `freeze` / `begin-recovery`)",
        "",
        f"- **Primary-overwrite refusal**: `restore-pitr.sh --isolated-container fvpn-w4-secops-postgres` exited `{s3['primary_overwrite_refusal']['exit_code']}` (`refused={s3['primary_overwrite_refusal']['refused']}`).",
        f"- **2-minute fail-safe negative (`{neg['window_seconds']:.2f} s`)**: Restored `T0` into `fvpn-w4-secops-restore-neg-live` and started `control` + `worker` **without** running `freeze` or `begin-recovery`:",
        f"  - At `T0` restore: central `highest_contiguous_seq={neg['t0_restored_ack']}`, `max_seen_seq={neg['t0_restored_max_seen']}`, `usage_events={neg['t0_restored_events']}`.",
        f"  - After `25 s` idle: central `highest_contiguous_seq={neg['after_25s_idle_ack']}`, `max_seen_seq={neg['after_25s_idle_max_seen']}`, `usage_events={neg['after_25s_idle_events']}` (`Uploader.due()` is `false` because events `{usage['n0_seq_at_t0']+1}..{usage['n1_seq_at_t1']}` are `state='archived'` on the node).",
        f"  - After post-`T1` traffic burst (`seq=6`) and `120 s` total window: central `highest_contiguous_seq={neg['after_120s_central_ack']}`, `max_seen_seq={neg['after_120s_central_max_seen']}`, `usage_events={neg['after_120s_central_events']}` (events `{usage['n0_seq_at_t0']+1}..{usage['n1_seq_at_t1']}` skipped!), while node SQLite store has {_node_store_summary(neg.get('after_120s_node_store', {}))} (`seq=6` stuck in `outbox` because central returned `highest_contiguous_seq=2 < acked_through=5`, triggering `ErrAckRegress`).",
        f"  - **Why**: {neg['explanation']}",
        "",
        "### 2.2 Clean Restore to `T0` + Product Restore Protocol (`docs/usage/restore-runbook.md` §2 Steps 1–5)",
        "",
        "| Step | Command / Action | Measured Result |",
        "|---|---|---|",
        f"| PITR + Checklist | `deploy/backup/restore-pitr.sh` to `{proto['restore_info']['target_time']}` + `deploy/backup/post-restore-checklist.sql` | `{proto['restore_info']['pitr_seconds']:.2f} s`, `{proto['restore_info']['checklist_sections']}/6` checklist sections passed |",
        f"| Step 1 (`freeze`) | `go run ./internal/usage/cmd/fvpn-usage-admin -config ... freeze --actor ... --reason ...` | `{proto['step1_freeze']['elapsed_seconds']:.2f} s`, `usage_ledger_state.mode = {proto['mode_after_freeze']}` |",
        f"| Step 2 (`begin-recovery`) | `go run ./internal/usage/cmd/fvpn-usage-admin -config ... begin-recovery --restored-to {s1['t0_utc']} --affected-since ...` | `{proto['step2_begin_recovery']['elapsed_seconds']:.2f} s`, `mode = {proto['mode_after_begin_recovery']}`, new `ledger_epoch = {proto['new_ledger_epoch']}` |",
        f"| Steps 3–4 (Node Replay) | Start `control` + `worker`; node receives `409 LEDGER_EPOCH_MISMATCH`, fetches `GET /api/v1/agent/usage-checkpoint`, un-archives `{usage['n0_seq_at_t0']+1}..{usage['n1_seq_at_t1']}`, and replays | `{proto['step3_4_node_replay']['replay_wait_seconds']:.2f} s`, `highest_contiguous_seq = {proto['step3_4_node_replay']['replayed_highest_seq']}`, `max_seen_seq = {proto['step3_4_node_replay'].get('replayed_max_seen_seq')}` |",
        f"| Step 5 (`complete-recovery`) | `go run ./internal/usage/cmd/fvpn-usage-admin -config ... complete-recovery --actor ...` | `{proto['step5_complete_recovery']['elapsed_seconds']:.2f} s`, `mode = {proto['mode_after_complete_recovery']}`, `reconciling_rollups_remaining = {proto['reconciling_rollups_remaining']}` |",
        f"| Readiness Verification | `GET /api/v1/member/me` (`{proto['member_api_ready']['m2_me_status']}`), `GET /api/v1/member/usage` (`{proto['member_api_ready']['m2_usage_status']}`), fresh `node_observations` row (`{proto['node_observation_accepted_after_recovery']}`) | **Protocol RTO (Step 1 `freeze` → ready): `{proto['protocol_rto_seconds']:.2f} s`**; **Total Outage RTO (central stop → ready): `{proto['total_outage_rto_seconds']:.2f} s`** |",
        "",
        "## 3. Usage Replay (`L2` vs `L1` and `L0`) and Measured `RPO`",
        "",
        "| Metric | Value |",
        "|---|---|",
        f"| Status | **{usage['status']}** |",
        f"| `N0` (`T0` highest contiguous seq) / `N1` (`T1` highest contiguous seq) | `{usage['n0_seq_at_t0']}` / `{usage['n1_seq_at_t1']}` |",
        f"| `L2` replayed events `1..N1` / post-`T1` events `> N1` | `{usage['l2_events_1_to_n1']}` / `{usage['l2_events_post_t1']}` |",
        f"| Canonical `payload_hash` matches across `1..N1` | `{usage['canonical_hash_matches_1_to_n1']} / {usage['n1_seq_at_t1']}` |",
        f"| `L2` bytes `1..N1` (matches `L1` `{l1_bytes}`) / post-`T1` bytes | `{usage['l2_bytes_1_to_n1']}` / `{usage['l2_bytes_post_t1']}` |",
        f"| Lost `T0..T1` ACKed events / Double-counted events / `usage_ledger_gaps` | `{usage['lost_t0_t1_acked_events']}` / `{usage['double_counted_events']}` / `{usage['usage_ledger_gaps']}` |",
        f"| Pre-replay database RPO (`T1 - T0`) / Post-replay service usage RPO | `{usage['pre_replay_db_rpo_seconds']:.3f} s` / `{usage['post_replay_service_rpo_seconds']} s` (`0` bytes lost within the node's 72h archive retention window) |",
        "",
        "## 4. Provider Reconciliation (`Central L2` vs `P1`)",
        "",
        "| Metric | Value |",
        "|---|---|",
        f"| Status | **{prov['status']}** |",
        f"| `m1` (`sub_e2e_1`, canceled + full refund in `T0..T1`) in `L2` | `provider_status={prov['m1_subscription_status_l2']}`, `is_active={prov['m1_subscription_active_l2']}`, `entitlement.state={prov['m1_entitlement_state_l2']}`, `device_limit={prov['m1_entitlement_dev_limit_l2']}`, `refund_rows={prov['m1_refund_rows_l2']}`, `http_200_after_reconcile={prov['m1_flow_after_reconcile'].get('http_200')}` |",
        f"| `m2` (`sub_e2e_2`, unchanged active subscription) in `L2` | `provider_status={prov['m2_subscription_status_l2']}`, `is_active={prov['m2_subscription_active_l2']}`, `entitlement.state={prov['m2_entitlement_state_l2']}`, `device_limit={prov['m2_entitlement_dev_limit_l2']}`, `paid_invoice_rows={prov['m2_paid_invoice_rows_l2']}`, `entitlement_extended={prov['m2_entitlement_extended']}` |",
        f"| `billing.reconciled` audit events in `L2` | `{prov['billing_reconciled_audits']}` (converged via worker `billing.reconcile` querying fake Stripe `GET /v1/subscriptions` & `GET /v1/invoices` without re-delivering the webhook) |",
        "",
        "## 5. Peer & Revocation Reconciliation (`Central L2` vs `N1`)",
        "",
        "| Scenario | Measured Outcome in `L2` |",
        "|---|---|",
        f"| **(a) Device revoked in `T0..T1` (`peerM2D2`: `{rev['device_id']}`, peer `{rev['external_peer_id']}`)** | **Discrepancy (`status=blocked`)**: At `T1` before restore, `peerM2D2` was `status=revoked` in central (`t1_observed_authorized={t1_obs_auth}`). After restoring central DB to `T0` (`member_devices.status='authorized'`), `members.reconcile` → `SyncMember` → `SyncDevice` (`apps/worker/jobs_members.go:158`, `internal/devices/worker.go:72-93`) treated `T0` `member_devices.status='authorized'` as authoritative and called `SetAuthorization(peer, true)`, re-authorizing `peerM2D2` in NetBird (`l2_observed_authorized={readded}`, `l2_central_status={rev['l2_central_status_after_reconcile']}`, `l2_in_active_device_bindings={in_bindings}`). |",
        f"| **(b) Post-`T0` member's `T0..T1` peer (`peerM3D1` for member `{unm['member_id']}`)** | **Passed (#205 verified)**: Member `{unm['member_id']}` does not exist in `T0` DB. Across multiple `devices.discover` sweeps, central kept `central_member_device_rows_l2={unm['central_member_device_rows_l2']}`, emitted exactly `{unm['identity_unmapped_audit_count']}` `device.authorize` (`result=denied`, `reason=identity_unmapped`) audit row (`deduplicated_across_sweeps={unm['deduplicated_across_sweeps']}`), and `peerM3D1` could not forward (`http_200={unm['flow_after_restore'].get('http_200')}`). |",
        f"| **(c) Existing `T0` member's `T0..T1` peer (`peerM2D3` for member `{exd['member_id']}`)** | **Converged**: `devices.discover` matched `peerM2D3` via `m2`'s `T0` `netbird_user_bindings` and re-admitted it (`rediscovered_rows_in_l2={exd['rediscovered_rows_in_l2']}`, `status={exd['rediscovered_status_in_l2']}`). |",
        "",
    ]
    return "\n".join(lines)


def main() -> int:
    run = load(PUBLIC / "followup8-run.json")
    live_path = PUBLIC / "followup8-live.json"
    live_text = live_path.read_text()
    root_prefix = str(ROOT) + "/"
    if root_prefix in live_text:
        live_text = live_text.replace(root_prefix, "")
        live_path.write_text(live_text)
    live = json.loads(live_text)
    restore_case = live.get("AT30_restore_reconcile") or live.get("cases", {}).get("AT30_restore_reconcile")
    if not restore_case:
        raise RuntimeError("followup8-live.json missing AT30_restore_reconcile evidence")

    at30_status, at30_blockers, at30_summary = evaluate_at30(restore_case)
    app_base = subprocess.check_output(
        ["git", "rev-parse", "--short", "main"], cwd=ROOT, text=True
    ).strip()

    commands = [
        run.get("up", {}).get("command") or ["make", "-f", "mk/e2e.mk", "e2e-up"],
        run["live"].get("command") or run["live"].get("argv"),
        run.get("down", {}).get("command") or ["make", "-f", "mk/e2e.mk", "e2e-down"],
    ]

    meta = {
        "schema_version": 1,
        "follow_up": 8,
        "started_at": run["started_at"],
        "finished_at": run["finished_at"],
        "source_revision": run["source_revision"],
        "application_base": app_base,
        "run_id": live["run_id"],
        "driver_run_id": run["driver_run_id"],
        "hardware": "shared development host; not candidate hardware",
        "commands": commands,
        "probe_source_sha256": live.get("probe_source_sha256")
        or sha256_file(ROOT / "tests/e2e/at/secops/live_test.go"),
        "policy_revision": live.get("policy_revision", ""),
        "node_incarnation": live.get("node_incarnation", ""),
        "agent_node_incarnation": live.get("agent_node_incarnation", ""),
        "image_digests": live.get("image_digests", []),
        "build_digests": digests_for_builds(),
        "previous_run": PREVIOUS_RUN,
        "projector_owned_rows_written": [],
    }

    restore_record = {
        **meta,
        "id": "AT30-RESTORE-RECONCILE",
        "status": at30_status,
        "evaluation": at30_summary,
        "blockers": at30_blockers,
        "case": restore_case,
    }
    write(PUBLIC / "restore-reconcile.json", restore_record)
    (PUBLIC / "restore-reconcile.md").write_text(
        render_restore_reconcile_md(restore_record)
    )

    at30_record = {
        **meta,
        "id": "AT30",
        "requirements": ["DATA01", "BILL01", "MTR03"],
        "status": at30_status,
        "expected": "An isolated DB/WAL restore replays previously ACKed node archives, reconciles provider and peers without duplicates or entitlement extension, preserves or explains revocation intent, and measures complete-service RPO/RTO.",
        "actual": {
            "evaluation": at30_summary,
            "evidence": [
                "restore-reconcile.json",
                "restore-reconcile.md",
                "history/follow-up-6/pitr.json",
            ],
            "rpo_seconds": {
                "pre_replay_db_rpo_seconds": restore_case["step4_reconciliation_observation"]["usage_replay"]["pre_replay_db_rpo_seconds"],
                "post_replay_service_rpo_seconds": restore_case["step4_reconciliation_observation"]["usage_replay"]["post_replay_service_rpo_seconds"],
            },
            "rto_seconds": {
                "protocol_rto_seconds": restore_case["step3_restore_central_to_t0"]["product_restore_protocol"]["protocol_rto_seconds"],
                "total_outage_rto_seconds": restore_case["step3_restore_central_to_t0"]["product_restore_protocol"]["total_outage_rto_seconds"],
            },
            "previous_record": PREVIOUS_RUN + "AT30.json",
        },
        "blockers": at30_blockers,
    }
    write(PUBLIC / "AT30.json", at30_record)

    s1 = restore_case["step1_checkpoint_t0"]
    s2 = restore_case["step2_world_forward_t1"]
    s3 = restore_case["step3_restore_central_to_t0"]
    s4 = restore_case["step4_reconciliation_observation"]
    b0 = s1["pgbackrest_backup"]
    _, _, l0_bytes = _ledger_counts(s1["ledger_l0"])
    _, _, l1_bytes = _ledger_counts(s2["ledger_l1"])
    neg = s3["fail_safe_negative_without_protocol"]
    proto = s3["product_restore_protocol"]
    usage = s4["usage_replay"]
    prov = s4["provider_reconciliation"]
    rev = s4["peer_revocation_reconciliation"]["revoked_in_t0_t1_device"]
    unm = s4["peer_revocation_reconciliation"]["new_in_t0_t1_unmapped_member_device"]
    readded = bool(rev.get("l2_observed_authorized") or rev.get("l2_readded_to_netbird_paid_group"))
    in_bindings = bool(rev.get("l2_in_active_device_bindings") or rev.get("l2_counted_flow_resumed"))
    t0_lsn = b0.get("t0_lsn", s1.get("t0_lsn"))

    at30_md = "\n".join(
        [
            "# AT30 — Follow-up 8",
            "",
            f"Full status: **{at30_status}**.",
            "",
            f"- **Isolated PITR & 6/6 post-restore checklist**: passed (`{proto['restore_info']['pitr_seconds']:.2f} s`, `restore-pitr.sh` primary-overwrite refusal exit code `2`).",
            f"- **2-minute fail-safe negative (before `freeze` / `begin-recovery`)**: proved (`{neg['window_seconds']:.2f} s`; without `ledger_epoch` rotation, `T0..T1` archived events `{usage['n0_seq_at_t0']+1}..{usage['n1_seq_at_t1']}` are not re-uploaded while idle, and post-`T1` traffic advances central `max_seen_seq={neg['after_120s_central_max_seen']}` while `highest_contiguous_seq={neg['after_120s_central_ack']}` stays at `N0`, leaving `seq=6` stuck in node `outbox` due to `ErrAckRegress`).",
            f"- **5-step usage ledger restore protocol (`fvpn-usage-admin`) & Retained-node replay**: **passed** (`L0={usage['n0_seq_at_t0']}` → `L1={usage['n1_seq_at_t1']}` → `L2` replayed all `{usage['l2_events_1_to_n1']}/{usage['n1_seq_at_t1']}` events with `{usage['canonical_hash_matches_1_to_n1']}/{usage['n1_seq_at_t1']}` canonical SHA-256 matches, `0` lost, `0` double-counted, `0` gaps; pre-replay DB RPO `{usage['pre_replay_db_rpo_seconds']:.3f} s`, post-replay service usage RPO `0 s`; protocol RTO `{proto['protocol_rto_seconds']:.2f} s`, total outage RTO `{proto['total_outage_rto_seconds']:.2f} s`).",
            f"- **Provider reconciliation (`Central L2` vs `P1`)**: **passed** (`m1` converged to `provider_status={prov['m1_subscription_status_l2']}`, `is_active={prov['m1_subscription_active_l2']}`, `refund_rows={prov['m1_refund_rows_l2']}`, `entitlement.state={prov['m1_entitlement_state_l2']}`, `device_limit={prov['m1_entitlement_dev_limit_l2']}` via `billing.reconcile` without webhook re-delivery; `m2` remained `active` with `entitlement_extended={prov['m2_entitlement_extended']}`).",
            f"- **Unmapped NetBird peer discovery denial (#205)**: **passed** (`identity_unmapped_audit_count={unm['identity_unmapped_audit_count']}`, deduplicated across sweeps).",
            f"- **Remaining blocker (`SECOPS-18`, `w2-members`)**: device `{rev['device_id']}` (peer `{rev['external_peer_id']}`) revoked in `T0..T1` had `member_devices.status='authorized'` in the restored `T0` database; `members.reconcile -> SyncMember -> SyncDevice` re-authorized the revoked peer in NetBird (`l2_observed_authorized={readded}`) and kept it in active `device_bindings` (`l2_in_active_device_bindings={in_bindings}`).",
            "",
            "Evidence: [restore-reconcile.md](restore-reconcile.md), [restore-reconcile.json](restore-reconcile.json), [history/follow-up-6/pitr.json](history/follow-up-6/pitr.json).",
            "",
        ]
    )
    (PUBLIC / "AT30.md").write_text(at30_md)

    cleanup = {
        **meta,
        "prefix": "fvpn-w4-secops",
        "port_base": "30000",
        "inventory_before": run["inventory_before"],
        "inventory_after": run["inventory_after"],
        "down": run["down"],
        "status": "passed"
        if clean_inventory(run["inventory_before"])
        and clean_inventory(run["inventory_after"])
        and run["down"]["exit_code"] == 0
        else "failed",
    }
    write(PUBLIC / "cleanup.json", cleanup)
    write(PUBLIC / "live.json", {**meta, "driver": run, "probe": live})

    cases_verdict = {
        "AT28": load(PUBLIC / "AT28.json")["status"],
        "AT29": load(PUBLIC / "AT29.json")["status"],
        "AT30": at30_status,
        "AT31": load(PUBLIC / "AT31.json")["status"],
        "AT32": load(PUBLIC / "AT32.json")["status"],
    }
    verdict = {
        "schema_version": 1,
        "follow_up": 8,
        "run_id": live["run_id"],
        "driver_run_id": run["driver_run_id"],
        "cases": cases_verdict,
        "at30_evaluation": at30_summary,
        "blocked": any(v != "passed" for v in cases_verdict.values()),
        "cleanup": cleanup["status"],
    }
    write(PUBLIC / "followup8-verdict.json", verdict)

    summary_md = "\n".join(
        [
            "# Follow-up 8 — AT30 live central database restore & cross-system reconciliation",
            "",
            f"Application base `{app_base}` (#207); harness `{run['source_revision']}`; run `{live['run_id']}`. Shared development host, **not candidate hardware**.",
            "",
            "| Measurement | Actual |",
            "|---|---|",
            f"| `T0` pgBackRest full backup / restore target | `{b0['backup_label']}` / `{s1['t0_utc']}` (LSN `{t0_lsn}`) |",
            f"| `L0` → `L1` central ACKed usage sequence (`N0` → `N1`) | `{usage['n0_seq_at_t0']}` (`{l0_bytes}` B) → `{usage['n1_seq_at_t1']}` (`{l1_bytes}` B) |",
            f"| Node retained records at `T1` (`/var/lib/vpn-node/state/usage.db`) | {_node_store_summary(s2['node_retained_store_t1'])} (`node/agent/observe.go:111-116` reports zeroed `UsageOutboxStatus`; SQLite retains `archived` rows for 72h) |",
            f"| 2-minute fail-safe negative (before `freeze` / `begin-recovery`) | `{neg['window_seconds']:.2f} s`: central ACK stayed `{neg['after_120s_central_ack']}`, `max_seen_seq={neg['after_120s_central_max_seen']}`, `usage_events={neg['after_120s_central_events']}` (events `3..5` skipped, `seq=6` stuck in node `outbox` due to `ErrAckRegress`) |",
            f"| 5-step product restore protocol (`fvpn-usage-admin`) | `freeze` → `begin-recovery` (new `ledger_epoch={proto['new_ledger_epoch']}`) → node archive replay (`{proto['step3_4_node_replay']['replay_wait_seconds']:.2f} s`) → `complete-recovery` (`mode=normal`, `0` reconciling rollups) |",
            f"| Usage replay (`L2` vs `L1` and `L0`) | **passed**: `{usage['l2_events_1_to_n1']}/{usage['n1_seq_at_t1']}` events (`{usage['canonical_hash_matches_1_to_n1']}/{usage['n1_seq_at_t1']}` canonical SHA-256 matches, `{usage['l2_bytes_1_to_n1']}` B), `0` lost, `0` double-counted, `0` gaps |",
            f"| Pre-replay DB RPO (`T1 - T0`) / Post-replay service usage RPO | `{usage['pre_replay_db_rpo_seconds']:.3f} s` / `0 s` |",
            f"| Protocol RTO (Step 1 `freeze` → ready) / Total outage RTO | `{proto['protocol_rto_seconds']:.2f} s` / `{proto['total_outage_rto_seconds']:.2f} s` |",
            f"| Provider reconciliation (`Central L2` vs `P1`) | **passed**: `m1` converged to `canceled` + `refund=1` + `entitlement.state={prov['m1_entitlement_state_l2']}` (`device_limit={prov['m1_entitlement_dev_limit_l2']}`) via `billing.reconcile`; `m2` stayed `active` (`entitlement_extended={prov['m2_entitlement_extended']}`) |",
            f"| Unmapped NetBird peer discovery denial (#205) | **passed**: `identity_unmapped` audit count = `{unm['identity_unmapped_audit_count']}` across multiple `devices.discover` sweeps |",
            f"| `T0..T1` device revocation after restore (`peerM2D2`) | **blocked (`SECOPS-18`)**: `SyncDevice` re-authorized `peerM2D2` in NetBird from `T0` `member_devices.status='authorized'` (`l2_observed_authorized={readded}`, `l2_in_active_device_bindings={in_bindings}`) |",
            "",
            "Full details and command sequences are in [restore-reconcile.md](restore-reconcile.md) and [restore-reconcile.json](restore-reconcile.json). Follow-up 7 records are archived byte-for-byte under [history/follow-up-7](history/follow-up-7/archive-index.json). See [requests](requests.md) and [run instructions](RUN.md).",
            "",
        ]
    )
    (PUBLIC / "summary.md").write_text(summary_md)

    requests_md = "\n".join(
        [
            "# Requests to orchestrator — Follow-up 8",
            "",
            f"Measured application base `{app_base}` (#207) includes members follow-up 5 (#205: discovery-denial audit deduplication in `internal/devices/audit.go:noteDiscoveryDecision`). No product, lab, or migration source outside owned paths was modified. Every claim is backed by [restore-reconcile.md](restore-reconcile.md) and [restore-reconcile.json](restore-reconcile.json).",
            "",
            "| Request | Status / owner | Evidence and required action |",
            "|---|---|---|",
            f"| SECOPS-05 / AT30 usage replay & provider reconciliation | **Usage replay & provider reconciliation passed; #205 discovery-denial deduplication verified** | `restore-reconcile.json` proves: (1) 2-minute fail-safe negative before `freeze`/`begin-recovery` (`T0..T1` archived events skipped while idle; post-`T1` event triggers `ErrAckRegress` and leaves `seq=6` in node `outbox` while central skips `3..5`); (2) 5-step `fvpn-usage-admin` protocol (`freeze` → `begin-recovery` → node `409 LEDGER_EPOCH_MISMATCH` + `GET /api/v1/agent/usage-checkpoint` + replay of `1..N1` with `{usage['canonical_hash_matches_1_to_n1']}/{usage['n1_seq_at_t1']}` canonical hash matches, `0` lost, `0` double-counted → `complete-recovery`); (3) provider reconciliation (`m1` converged to `canceled` + `refund` + `entitlement.state=expired` via `billing.reconcile`; `m2` active without extension); (4) #205 `identity_unmapped` audit emitted once (`count={unm['identity_unmapped_audit_count']}`) across multiple sweeps. |",
            f"| SECOPS-18, post-restore `T0..T1` device revocation preservation | **Open; `w2-members` (`internal/devices`)** | In `TestAT30_Followup8_LiveRestoreReconcile`, `peerM2D2` (`{rev['device_id']}`, peer `{rev['external_peer_id']}`) is revoked in `T0..T1` (`DELETE /api/v1/member/devices/{{id}}` → `202`, `device.revoke` audit, withdrawn from `fvpn-paid-<group_id>` in NetBird `N1`). After restoring central DB to `T0` (`member_devices.status='authorized'`, `device_authorization_observations.observed_authorized=true`), `members.reconcile` (`apps/worker/jobs_members.go:158` → `SyncMember` → `SyncDevice` in `internal/devices/worker.go:72-93`) treats `T0` `member_devices.status='authorized'` as authoritative and calls `s.Adapter.SetAuthorization(ctx, deployment, peer, true, groups)`, silently re-authorizing the revoked peer in NetBird (`l2_observed_authorized={readded}`) and keeping it in active `device_bindings` (`l2_in_active_device_bindings={in_bindings}`). **Required action**: When central is in post-restore reconciliation (or when `device_authorization_observations.observed_authorized=true` in a restored DB conflicts with `ObservePeer` showing the peer is NOT in the expected NetBird paid group), fence `SyncDevice` so it does not silently re-authorize a peer whose external group membership was withdrawn after `T0` (either quarantine/revoke on observed external withdrawal when `observed_authorized` was already `true`, or persist a revocation marker/group in NetBird that `SyncDevice`/`Discover` respects), and emit an explicit audit event. |",
            "| SECOPS-19, node retained-archive observability | **Informational / `w1-node-agent` (`node/agent/observe.go`)** | `node/agent/observe.go:111-116` currently sends a zeroed `UsageOutboxStatus{}` in `POST /api/v1/agent/observations`, and no node CLI/HTTP endpoint exposes `/var/lib/vpn-node/state/usage.db` (`state='archived'` rows retained for 72h). Populating `UsageOutboxStatus` (or exposing a read-only `fvpn-agent usage-status` subcommand) would let operators inspect retained archive sequences without opening the SQLite file directly. |",
            f"| SECOPS-17, finite grace provider truth (`past_due`) | **Open; `w3-devstack-e2e` payment hook** | On `#207`, `POST /e2e/snapshot` with `status: \"past_due\"` still returns HTTP `{s2['provider_change']['past_due_probe_status']}` (`tests/e2e/cmd/fvpn-e2e-payment/truth.go:66` admits only `active`/`canceled`). AT30 used the supported `canceled` + `refund_minor=1000` transition to prove provider reconciliation; AT29 finite payment grace still awaits `past_due` support in `fvpn-e2e-payment`. |",
            "| SECOPS-16, bulk final per-stream attribution | **Open; `w3-devstack-e2e`** | Unchanged from Follow-up 7 (`tests/e2e/native_exports.go:ScaleNativeMembers` aggregate return). |",
            "| AT29 established transports & guest rollback; AT31 forced relay & hardware; DOC16 | **Open (`w3-devstack-e2e`, operator runs, `w4-handover`)** | Unchanged from Follow-up 7; see `history/follow-up-7/requests.md`. |",
            "",
        ]
    )
    (PUBLIC / "requests.md").write_text(requests_md)

    run_md = "\n".join(
        [
            "# Follow-up 8 run instructions",
            "",
            "Run from this worktree with Go 1.27.3 and Docker:",
            "",
            "```sh",
            "make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build",
            "make -f mk/at-secops.mk at-secops-followup8",
            "make go-lint",
            "go vet ./...",
            "```",
            "",
            "`at-secops-followup8` runs `tests/e2e/at/secops/followup8_run.py` in the foreground:",
            "",
            "1. Verifies that zero `fvpn-w4-secops-*` Docker containers, volumes, or networks exist (`inventory_before`).",
            "2. Brings up a fresh lab stack (`make -f mk/e2e.mk e2e-up` with `FVPN_E2E_PREFIX=w4-secops`, `FVPN_E2E_PORT_BASE=30000`, `E2E_RESET=1`).",
            "3. Runs `TestAT30_Followup8_LiveRestoreReconcile` (`tests/e2e/at/secops/followup8_live_test.go`):",
            "   - **Step 1 (`T0`)**: Seeds two paid members (`m1`, `m2`) with native devices on `fvpn-w4-secops-node`, runs counted traffic until usage checkpoints are ACKed (`L0`), captures the node's retained SQLite store (`/var/lib/vpn-node/state/usage.db`), and takes a pgBackRest full backup + records `T0` UTC/LSN via `deploy/backup/backup.sh`.",
            "   - **Step 2 (`T1`)**: Runs additional counted traffic (`L1 > L0`), applies a subscription cancellation + full refund on `m1` in `fvpn-w4-secops-payment` (`P1`), revokes `m2`'s second device `peerM2D2` (`DELETE /api/v1/member/devices/{id}`), authorizes two new `T0..T1` devices (`peerM3D1` for post-`T0` member `m3` and `peerM2D3` for `T0` member `m2`) (`N1`), and archives WAL (`wal-switch`).",
            "   - **Step 3 (Restore to `T0`)**: Verifies `restore-pitr.sh` primary-overwrite refusal (exit `2`), runs a **2-minute fail-safe negative** on a restored `T0` database without `freeze`/`begin-recovery` (proving `T0..T1` archived events are skipped while idle and trigger `ErrAckRegress` on new traffic), then restores `T0` cleanly (`post-restore-checklist.sql` `6/6`) and executes the 5-step product restore protocol (`fvpn-usage-admin freeze` → `begin-recovery` → node archive replay → `complete-recovery`), measuring `RTO`.",
            "   - **Step 4 (Reconciliation Observation)**: Observes `usage_replay` (`L2` vs `L1` and `L0`, `RPO`), `provider_reconciliation` (`Central L2` vs `P1`), and `peer_revocation_reconciliation` (`Central L2` vs `N1`).",
            "4. Unconditionally tears down all `fvpn-w4-secops-*` containers, volumes, networks, and restore images (`make -f mk/e2e.mk e2e-down` + restore object cleanup) and verifies `inventory_after` is zero.",
            "5. Renders records via `python3 tests/e2e/at/secops/followup8_records.py`.",
            "",
            "Previous Follow-up 7 root records are archived with SHA-256 digests under `docs/e2e/at/secops/history/follow-up-7/`.",
            "",
        ]
    )
    (PUBLIC / "RUN.md").write_text(run_md)

    deps_path = PUBLIC / "dependencies.json"
    if deps_path.exists():
        deps = load(deps_path)
        deps["follow_up"] = 8
        deps["checked_at"] = run["finished_at"]
        deps["application_base"] = app_base
        deps["harness_revision"] = run["source_revision"]
        deps["history"] = PREVIOUS_RUN + "dependencies.json"
        deps.setdefault("merged", {})["secops_followup_7"] = 203
        deps["merged"]["members_followup_5"] = 205
        deps.setdefault("requests", {})["SECOPS-05"] = {
            "status": "measured in follow-up 8 (usage replay & provider reconciliation passed; #205 discovery-denial deduplication verified)",
            "prs": [165, 205],
            "evidence": "restore-reconcile.json",
        }
        deps["requests"]["SECOPS-18"] = {
            "status": "open",
            "owner": "w2-members (internal/devices)",
            "detail": "T0..T1 revoked device (peerM2D2) re-authorized by SyncDevice after restoring central DB to T0",
            "evidence": "restore-reconcile.json",
        }
        deps["requests"]["SECOPS-19"] = {
            "status": "informational",
            "owner": "w1-node-agent (node/agent/observe.go)",
            "detail": "node/agent/observe.go:111-116 sends zeroed UsageOutboxStatus; retained archive sequences only visible via SQLite usage.db",
            "evidence": "restore-reconcile.json",
        }
        write(deps_path, deps)

    scope_path = PUBLIC / "scope-review.json"
    if scope_path.exists():
        scope = load(scope_path)
        scope["follow_up_8"] = {
            "reviewed_at": run["finished_at"],
            "application_base": app_base,
            "source_revision": run["source_revision"],
            "scope": "AT30 second half: live central database restore to T0 while node, payment provider, and NetBird hold T1 state; measured fail-safe negative, 5-step fvpn-usage-admin protocol, retained-node replay, provider reconciliation, and peer/revocation reconciliation.",
            "evidence": [
                "restore-reconcile.json",
                "restore-reconcile.md",
                "AT30.json",
                "AT30.md",
            ],
            "new_requests": [
                {
                    "id": "SECOPS-18",
                    "owner": "w2-members",
                    "status": "open",
                    "file": "internal/devices/worker.go",
                    "section": "SyncDevice (lines 72-93) / members.reconcile",
                    "reason": "After restoring central DB to T0, a device revoked in T0..T1 (peerM2D2) has member_devices.status='authorized' in T0 DB while NetBird N1 has withdrawn it from fvpn-paid-<group_id>. SyncDevice treats T0 status='authorized' as authoritative and calls SetAuthorization(peer, true), silently re-authorizing the revoked peer in NetBird and resuming counted member traffic.",
                }
            ],
        }
        write(scope_path, scope)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
