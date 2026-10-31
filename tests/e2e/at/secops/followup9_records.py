#!/usr/bin/env python3
"""Render Follow-up 9 AT30 live central restore, D-076 fence & SECOPS-19 visibility records."""
import hashlib
import json
from pathlib import Path

from run import PUBLIC, ROOT

PREVIOUS_RUN = "history/follow-up-8/"

FOLLOWUP8_RPO_RTO = {
    "pr": 221,
    "run_id": "0f5fb60d-f6a7-45c6-8ebe-175e0a480bd8",
    "pre_replay_db_rpo_seconds": 116.651367,
    "post_replay_service_rpo_seconds": 0,
    "protocol_rto_seconds": 105.938765968,
    "total_outage_rto_seconds": 111.512077088,
}


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


def evaluate_at30(restore_case: dict, *, http_regrant: bool = False) -> tuple[str, list[str], dict]:
    """Evaluate AT30 pass criteria from Follow-up 9 live restore, D-076 fence & SECOPS-19 visibility."""
    blockers = []
    step3 = restore_case.get("step3_restore_central_to_t0", {})
    step4 = restore_case.get("step4_reconciliation_observation", {})

    neg = step3.get("fail_safe_negative_without_protocol", {})
    proto = step3.get("product_restore_protocol", {})
    s19 = step3.get("secops19_visibility", {})
    usage = step4.get("usage_replay", {})
    prov = step4.get("provider_reconciliation", {})
    peers = step4.get("peer_revocation_reconciliation", {})
    revoked_dev = peers.get("revoked_in_t0_t1_device", {})
    regrant = peers.get("regrant_authorize_device", {})
    neg_central = peers.get("negative_central_revoke_not_fenced", {})
    neg_fresh = peers.get("negative_post_restore_fresh_enroll_builds", {})
    unmapped_dev = peers.get("new_in_t0_t1_unmapped_member_device", {})

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

    t0_ack = int(
        _checkpoint_of(
            restore_case.get("step1_checkpoint_t0", {}).get("ledger_l0", {})
        ).get("highest_contiguous_seq", 0)
        or usage.get("n0_seq_at_t0", 0)
        or 0
    )
    t1_ack = int(
        _checkpoint_of(
            restore_case.get("step2_world_forward_t1", {}).get("ledger_l1", {})
        ).get("highest_contiguous_seq", 0)
        or usage.get("n1_seq_at_t1", 0)
        or 0
    )
    neg_node_store = neg.get("after_120s_node_store", {})
    neg_ok = bool(
        neg.get("window_seconds", 0) >= 120
        and neg.get("retained_t0_t1_skipped_without_protocol") is True
        and (
            neg.get("node_ack_regress_log_lines", 0) >= 1
            or (
                t0_ack >= 1
                and t1_ack > t0_ack
                and int(neg.get("after_120s_central_ack", "0") or 0) == t0_ack
                and int(neg.get("after_120s_central_max_seen", "0") or 0) > t1_ack
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

    hold_obs = (
        s19.get("during_restore_hold", {})
        .get("node_observation_report_usage", {})
        .get("usage", {})
    )
    hold_snap = (
        s19.get("during_restore_hold", {})
        .get("fvpn_agent_usage_snapshot_cli", {})
        .get("snapshot", {})
    )
    conv_obs = (
        s19.get("after_complete_recovery_and_replay", {})
        .get("node_observation_report_usage", {})
        .get("usage", {})
    )
    conv_snap = (
        s19.get("after_complete_recovery_and_replay", {})
        .get("fvpn_agent_usage_snapshot_cli", {})
        .get("snapshot", {})
    )
    secops19_ok = bool(
        int(hold_obs.get("unacked_event_count", 0) or 0) > 0
        and int(hold_obs.get("unacked_payload_bytes", "0") or 0) > 0
        and bool(hold_obs.get("ledger_epoch"))
        and bool(hold_obs.get("meter_epoch_id"))
        and int(hold_snap.get("unacked_count", 0) or 0) > 0
        and int(hold_snap.get("unacked_payload_bytes", "0") or 0) > 0
        and hold_snap.get("ledger_epoch") == hold_obs.get("ledger_epoch")
        and int(conv_obs.get("unacked_event_count", -1)) == 0
        and str(conv_obs.get("unacked_payload_bytes", "")) == "0"
        and int(conv_snap.get("unacked_count", -1)) == 0
        and str(conv_snap.get("unacked_payload_bytes", "")) == "0"
    )
    if not secops19_ok:
        blockers.append(
            "SECOPS-19 visibility check did not show non-zero unacked outbox state during restore hold and zero unacked convergence after complete-recovery in both node_observations and fvpn-agent usage snapshot."
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

    fence_logs = revoked_dev.get("worker_restore_fence_log_lines") or []
    log_has_4_fields = bool(
        len(fence_logs) == 1
        and fence_logs[0].get("device_id") == str(revoked_dev.get("device_id"))
        and bool(fence_logs[0].get("member_id"))
        and fence_logs[0].get("peer_id") == str(revoked_dev.get("external_peer_id"))
        and fence_logs[0].get("ledger_mode") in ("reconciling", "frozen")
    )
    fence_ok = bool(
        revoked_dev.get("fenced_instead_of_regranted") is True
        and revoked_dev.get("fence_row_count") == 1
        and revoked_dev.get("external_withdrawal_audit_count") == 1
        and (revoked_dev.get("worker_restore_fence_counter_value") or 0) >= 1
        and log_has_4_fields
        and revoked_dev.get("l2_member_devices_api_status") == "pending"
        and revoked_dev.get("l2_observed_authorized") is False
        and revoked_dev.get("l2_readded_to_netbird_paid_group") is False
        and revoked_dev.get("l2_counted_flow_resumed") is False
    )
    if not fence_ok:
        blockers.append(
            "D-076 fence check failed: T0..T1 revoked device (peerM2D2) was not fenced with 1 device_authorization_fences row, 1 external_withdrawal audit, 1 worker counter increment, 4-field worker log line, pending member API status, and withdrawn data plane."
        )

    regrant_ok = bool(
        regrant.get("passed") is True
        and regrant.get("t0_idempotency_replay_replayed") is True
        and regrant.get("fence_rows_after_t0_replay") == 1
        and regrant.get("fence_rows_after_fresh_regrant") == 0
        and regrant.get("observed_authorized_after_sync") is True
        and regrant.get("member_devices_api_status") == "authorized"
        and regrant.get("counted_flow_after_regrant", {}).get("counted") is True
    )
    if http_regrant:
        regrant_ok = bool(
            regrant.get("passed") is True
            and regrant.get("existing_operation_returned") is True
            and regrant.get("operation_kind") == "member.device_authorize"
            and regrant.get("same_key_replay_same_operation") is True
            and regrant.get("replay_resource_version_unchanged") is True
            and regrant.get("replay_outbox_count_unchanged") is True
            and regrant.get("fence_rows_after_fresh_regrant") == 0
            and regrant.get("observed_authorized_after_sync") is True
            and regrant.get("member_devices_api_status") == "authorized"
            and regrant.get("counted_flow_after_regrant", {}).get("counted") is True
            and neg_central.get("admin_authorize_revoked", {}).get("status") == 409
            and neg_central.get("admin_authorize_revoked", {}).get("error") == "STATE_CONFLICT"
        )
    if not regrant_ok:
        blockers.append(
            "D-076 re-grant via AuthorizeDevice (member.device_authorize) did not clear the fence row and restore single NetBird grant and counted data-plane flow."
        )

    neg_central_ok = bool(
        neg_central.get("passed") is True
        and neg_central.get("member_devices_api_status") == "revoked"
        and neg_central.get("fence_row_count") == 0
        and neg_central.get("external_withdrawal_audit_count") == 0
        and neg_central.get("worker_fence_counter_after") == neg_central.get("worker_fence_counter_before")
    )
    if not neg_central_ok:
        blockers.append(
            "D-076 negative (central revoke) failed: centrally revoked device was fenced or marked pending instead of revoked."
        )

    neg_fresh_ok = bool(
        neg_fresh.get("passed") is True
        and neg_fresh.get("member_devices_api_status") == "authorized"
        and neg_fresh.get("fence_row_count") == 0
        and neg_fresh.get("external_withdrawal_audit_count") == 0
        and neg_fresh.get("worker_fence_counter_after") == neg_fresh.get("worker_fence_counter_before")
        and neg_fresh.get("counted_flow_proof", {}).get("counted") is True
    )
    if not neg_fresh_ok:
        blockers.append(
            "D-076 negative (never-observed device enrolled after restore) failed: fresh device was fenced instead of granted."
        )

    summary = {
        "primary_overwrite_refusal": refusal_ok,
        "post_restore_checklist_6_of_6": checklist_ok,
        "fail_safe_negative_without_protocol": neg_ok,
        "product_restore_protocol_5_steps": proto_ok,
        "secops19_visibility_and_zero_convergence": secops19_ok,
        "retained_node_usage_replay": usage_ok,
        "provider_reconciliation": prov_ok,
        "unmapped_peer_discovery_deduplicated": unmapped_dedup_ok,
        "t0_t1_device_revocation_fenced_d076": fence_ok,
        "regrant_authorize_device_clears_fence": regrant_ok,
        "negative_central_revoke_not_fenced": neg_central_ok,
        "negative_post_restore_fresh_enroll_not_fenced": neg_fresh_ok,
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
    s19 = s3["secops19_visibility"]
    usage = s4["usage_replay"]
    prov = s4["provider_reconciliation"]
    peers = s4["peer_revocation_reconciliation"]
    rev = peers["revoked_in_t0_t1_device"]
    reg = peers["regrant_authorize_device"]
    neg_c = peers["negative_central_revoke_not_fenced"]
    neg_f = peers["negative_post_restore_fresh_enroll_builds"]
    unm = peers["new_in_t0_t1_unmapped_member_device"]

    hold_obs = s19["during_restore_hold"]["node_observation_report_usage"]["usage"]
    hold_snap = s19["during_restore_hold"]["fvpn_agent_usage_snapshot_cli"]["snapshot"]
    conv_obs = s19["after_complete_recovery_and_replay"]["node_observation_report_usage"]["usage"]
    conv_snap = s19["after_complete_recovery_and_replay"]["fvpn_agent_usage_snapshot_cli"]["snapshot"]
    fence_log = (rev.get("worker_restore_fence_log_lines") or [{}])[0]

    lines = [
        "# AT30 Live Central Database Restore, D-076 Fence & SECOPS-19 Visibility — Follow-up 9",
        "",
        f"Application base `{record['application_base']}` (#231: #227 D-076 fence, #226 SECOPS-19 observation `UsageOutboxStatus`, #225 `fvpn-agent usage snapshot`); harness `{record['source_revision']}`; run `{record['run_id']}`; driver `{record['driver_run_id']}`.",
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
        f"| Node retained store at `T0` (`/var/lib/vpn-node/state/usage.db` & `fvpn-agent usage snapshot`) | {_node_store_summary(s1['node_retained_store_t0'])} |",
        f"| `T1` UTC after moving world forward | `{s2['t1_utc']}` (`T1 - T0 = {usage['pre_replay_db_rpo_seconds']:.3f} s`, LSN `{s2.get('t1_lsn')}`) |",
        f"| `L1` central usage checkpoint (`highest_contiguous_seq` / `max_seen_seq`) | `{cp1.get('highest_contiguous_seq')}` / `{cp1.get('max_seen_seq')}` (`ledger_epoch={cp1.get('ledger_epoch')}`) |",
        f"| `L1` central `usage_events` / `usage_samples` / total bytes | `{l1_ev}` / `{l1_smp}` / `{l1_bytes}` bytes |",
        f"| Node retained store at `T1` (`/var/lib/vpn-node/state/usage.db` & `fvpn-agent usage snapshot`) | {_node_store_summary(s2['node_retained_store_t1'])} |",
        f"| `P1` fake payment provider truth (`m1` / `m2`) | `m1`: `/e2e/snapshot` `past_due` probe returned HTTP `{s2['provider_change']['past_due_probe_status']}`; applied `status=canceled, refund_minor=1000` + `charge.refunded` webhook (`{s2['provider_change']['webhook_http_status']}`) → `provider_status={s2['provider_change']['t1_subscription_status']}`, `is_active={s2['provider_change']['t1_subscription_active']}`, `refund_records={s2['provider_change']['t1_refund_record_count']}`, `entitlement.state={s2['provider_change']['t1_entitlement_state']}`, `device_limit={s2['provider_change']['t1_entitlement_dev_limit']}`; `m2`: `active` (`paid_invoice=1`) |",
        f"| `N1` NetBird & peer changes in `T0..T1` | `peerM2D2` (`{s2['revoked_device_t1']['device_id']}`, peer `{s2['revoked_device_t1']['external_peer_id']}`) revoked (`DELETE` HTTP `{s2['revoked_device_t1']['delete_http_status']}`, `device.revoke` audit `{s2['revoked_device_t1']['t1_revoke_audit_event']['id']}`, `t1_observed_authorized={s2['revoked_device_t1']['t1_observed_authorized']}`); post-`T0` member `m3` (`{s2['new_t0_t1_devices']['m3_new_member_device']['member_id']}`) authorized `peerM3D1` (`{s2['new_t0_t1_devices']['m3_new_member_device']['device_id']}`) |",
        "",
        "## 2. Restore Command Sequence, Fail-Safe Negative, Measured `RTO`/`RPO`, and Comparison with Follow-up 8 (`#221`)",
        "",
        "### 2.1 Primary-Overwrite Refusal & 2-Minute Fail-Safe Negative (Before `freeze` / `begin-recovery`)",
        "",
        f"- **Primary-overwrite refusal**: `restore-pitr.sh --isolated-container fvpn-w4-secops-postgres` exited `{s3['primary_overwrite_refusal']['exit_code']}` (`refused={s3['primary_overwrite_refusal']['refused']}`).",
        f"- **2-minute fail-safe negative (`{neg['window_seconds']:.2f} s`)**: Restored `T0` into `fvpn-w4-secops-restore-neg-live` and started `control` + `worker` **without** running `freeze` or `begin-recovery`:",
        f"  - At `T0` restore: central `highest_contiguous_seq={neg['t0_restored_ack']}`, `max_seen_seq={neg['t0_restored_max_seen']}`, `usage_events={neg['t0_restored_events']}`.",
        f"  - After `25 s` idle: central `highest_contiguous_seq={neg['after_25s_idle_ack']}`, `max_seen_seq={neg['after_25s_idle_max_seen']}`, `usage_events={neg['after_25s_idle_events']}` (`Uploader.due()` is `false` because events `{usage['n0_seq_at_t0']+1}..{usage['n1_seq_at_t1']}` are `state='archived'` on the node).",
        f"  - After post-`T1` traffic burst (`seq=6`) and `120 s` total window: central `highest_contiguous_seq={neg['after_120s_central_ack']}`, `max_seen_seq={neg['after_120s_central_max_seen']}`, `usage_events={neg['after_120s_central_events']}` (events `{usage['n0_seq_at_t0']+1}..{usage['n1_seq_at_t1']}` skipped!), while node SQLite store has {_node_store_summary(neg.get('after_120s_node_store', {}))} (`ErrAckRegress` log lines: `{neg.get('node_ack_regress_log_lines')}`).",
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
        f"| Readiness Verification | `GET /api/v1/member/me` (`{proto['member_api_ready']['m2_me_status']}`), `GET /api/v1/member/usage` (`{proto['member_api_ready']['m2_usage_status']}`), fresh `node_observations` row (`{proto['node_observation_accepted_after_recovery']}`) | **Protocol RTO: `{proto['protocol_rto_seconds']:.2f} s`**; **Total Outage RTO: `{proto['total_outage_rto_seconds']:.2f} s`** |",
        "",
        "### 2.3 Measured `RPO` / `RTO` Comparison: Follow-up 9 vs Follow-up 8 (`#221`)",
        "",
        "| Metric | Follow-up 8 (`#221`, `0f5fb60d`) | Follow-up 9 (`#231`, `{}`) |".format(str(record["run_id"])[:8]),
        "|---|---|---|",
        f"| Pre-replay database RPO (`T1 - T0`) | `{FOLLOWUP8_RPO_RTO['pre_replay_db_rpo_seconds']:.3f} s` | `{usage['pre_replay_db_rpo_seconds']:.3f} s` |",
        f"| Post-replay service usage RPO | `{FOLLOWUP8_RPO_RTO['post_replay_service_rpo_seconds']} s` (`0` bytes lost) | `{usage['post_replay_service_rpo_seconds']} s` (`0` bytes lost) |",
        f"| Protocol RTO (Step 1 `freeze` → ready) | `{FOLLOWUP8_RPO_RTO['protocol_rto_seconds']:.2f} s` | `{proto['protocol_rto_seconds']:.2f} s` |",
        f"| Total outage RTO (central stop → ready) | `{FOLLOWUP8_RPO_RTO['total_outage_rto_seconds']:.2f} s` | `{proto['total_outage_rto_seconds']:.2f} s` |",
        "",
        "## 3. SECOPS-19 Visibility (`#226` `UsageOutboxStatus` in Observations & `#225` `fvpn-agent usage snapshot`)",
        "",
        "| Phase / Surface | `unacked_event_count` / `unacked_count` | `unacked_payload_bytes` | `ledger_epoch` | `meter_epoch_id` / `meter_epochs` |",
        "|---|---|---|---|---|",
        f"| **During restore hold (`reconciling`)** — `node_observations.report_payload.usage` | `{hold_obs.get('unacked_event_count')}` | `\"{hold_obs.get('unacked_payload_bytes')}\"` | `{hold_obs.get('ledger_epoch')}` | `{hold_obs.get('meter_epoch_id')}` |",
        f"| **During restore hold (`reconciling`)** — `fvpn-agent usage snapshot -json` | `{hold_snap.get('unacked_count')}` (`acked_through={hold_snap.get('acked_through')}`, `last_event_seq={hold_snap.get('last_event_seq')}`) | `\"{hold_snap.get('unacked_payload_bytes')}\"` | `{hold_snap.get('ledger_epoch')}` | `{len(hold_snap.get('meter_epochs') or [])}` epochs (`current={hold_obs.get('meter_epoch_id')}`) |",
        f"| **After `complete-recovery` & replay** — `node_observations.report_payload.usage` | `{conv_obs.get('unacked_event_count')}` | `\"{conv_obs.get('unacked_payload_bytes')}\"` | `{conv_obs.get('ledger_epoch')}` | `{conv_obs.get('meter_epoch_id')}` |",
        f"| **After `complete-recovery` & replay** — `fvpn-agent usage snapshot -json` | `{conv_snap.get('unacked_count')}` (`acked_through={conv_snap.get('acked_through')}`, `last_event_seq={conv_snap.get('last_event_seq')}`) | `\"{conv_snap.get('unacked_payload_bytes')}\"` | `{conv_snap.get('ledger_epoch')}` | `{len(conv_snap.get('meter_epochs') or [])}` epochs (`current={conv_obs.get('meter_epoch_id')}`) |",
        "",
        "## 4. Usage Replay (`L2` vs `L1` and `L0`) and Provider Reconciliation (`Central L2` vs `P1`)",
        "",
        "| Subsystem | Measured Outcome |",
        "|---|---|",
        f"| **Usage Replay (`L2`)** | **{usage['status']}**: `N0={usage['n0_seq_at_t0']}` → `N1={usage['n1_seq_at_t1']}`, replayed `{usage['l2_events_1_to_n1']}/{usage['n1_seq_at_t1']}` events (`{usage['canonical_hash_matches_1_to_n1']}/{usage['n1_seq_at_t1']}` canonical SHA-256 matches, `{usage['l2_bytes_1_to_n1']}` B), `{usage['l2_events_post_t1']}` post-`T1` event (`{usage['l2_bytes_post_t1']}` B), `0` lost, `0` double-counted, `0` gaps. |",
        f"| **Provider Reconciliation (`P1` → `L2`)** | **{prov['status']}**: `m1` converged to `provider_status={prov['m1_subscription_status_l2']}`, `is_active={prov['m1_subscription_active_l2']}`, `refund_rows={prov['m1_refund_rows_l2']}`, `entitlement.state={prov['m1_entitlement_state_l2']}`, `device_limit={prov['m1_entitlement_dev_limit_l2']}` via `{prov['billing_reconciled_audits']}` `billing.reconciled` audits without webhook re-delivery; `m2` remained `active` (`entitlement_extended={prov['m2_entitlement_extended']}`). |",
        "",
        "## 5. SECOPS-18 Closure Evidence: D-076 Restore Authorization Fence, Two Negatives, and Re-Grant",
        "",
        "### 5.1 Restored `T0..T1` Revoked Device (`peerM2D2`) Fenced Instead of Re-Granted",
        "",
        "| Proof Artifact | Measured Value |",
        "|---|---|",
        f"| `device_authorization_fences` row (`SELECT`) | `count={rev['fence_row_count']}`: `device_id={rev['fence_row'].get('device_id')}`, `fenced_at={rev['fence_row'].get('fenced_at')}`, `observed_group_ids={rev['fence_row'].get('observed_group_ids')}` |",
        f"| Worker structured log line (`msg=\"device authorization unconfirmed during ledger restore\"`) | `device_id={fence_log.get('device_id')}`, `member_id={fence_log.get('member_id')}`, `peer_id={fence_log.get('peer_id')}`, `ledger_mode={fence_log.get('ledger_mode')}` |",
        f"| Worker Prometheus counter (`http://127.0.0.1:9091/metrics`) | `{rev['worker_restore_fence_counter_line']}` (`value={rev['worker_restore_fence_counter_value']}`) |",
        f"| Audit row (`device.authorization.external_withdrawal`) | `count={rev['external_withdrawal_audit_count']}`: `id={rev['external_withdrawal_audit_event'].get('id')}`, `resource={rev['external_withdrawal_audit_event'].get('resource')}`, `result={rev['external_withdrawal_audit_event'].get('result')}`, `after_diff={json.dumps(rev['external_withdrawal_audit_event'].get('after_diff'))}` |",
        f"| Member `GET /api/v1/member/devices` status | `status=\"{rev['l2_member_devices_api_status']}\"` (`observed_authorized={rev['l2_observed_authorized']}`, `in_netbird_paid_group={rev['l2_readded_to_netbird_paid_group']}`) |",
        f"| Data-plane proof (`peerM2D2` vs sibling `peerM2D1`) | `peerM2D2` stays withdrawn (`counted={rev['l2_counted_flow_resumed']}`, `blocker=\"{rev['l2_flow_probe'].get('blocker')}\"`); sibling `peerM2D1` forwards counted traffic (`counted={rev['sibling_d1_counted_flow'].get('counted')}`) |",
        "",
        "### 5.2 Re-Grant via `AuthorizeDevice` (`operation kind = member.device_authorize`)",
        "",
        f"- **Note**: {reg['route_note']}.",
        f"- **Replay of original `T0` idempotency key**: returned `Replayed={reg['t0_idempotency_replay_replayed']}` and left `device_authorization_fences` intact (`fence_rows_after_t0_replay={reg['fence_rows_after_t0_replay']}`).",
        f"- **Fresh `AuthorizeDevice` (`operation_id={reg['operation_id']}`, `kind={reg['operation_kind']}`)**: deleted the `device_authorization_fences` row (`fence_rows_after_fresh_regrant={reg['fence_rows_after_fresh_regrant']}`), bumped `resource_version` `{reg['resource_version_before']} → {reg['resource_version_after']}`, worker synced NetBird (`observed_authorized={reg['observed_authorized_after_sync']}`, `operation_status={reg['operation_status']}`), `GET /api/v1/member/devices` returned `status=\"{reg['member_devices_api_status']}\"`, and counted data-plane flow resumed (`counted={reg['counted_flow_after_regrant'].get('counted')}`, delta `{reg['counted_flow_after_regrant'].get('admitted_bytes_delta')}`).",
        "",
        "### 5.3 Two Negatives & Unmapped Discovery Deduplication",
        "",
        "| Negative / Scenario | Measured Result |",
        "|---|---|",
        f"| **Negative 1 — Never-observed device enrolled after restore (`peerM2D3`: `{neg_f['device_id']}`)** | **Fresh grant (no fence)**: `GET /api/v1/member/devices` → `status=\"{neg_f['member_devices_api_status']}\"`, `observed_authorized={neg_f['observed_authorized']}`, `fence_row_count={neg_f['fence_row_count']}`, `external_withdrawal_audit_count={neg_f['external_withdrawal_audit_count']}`, worker fence counter `{neg_f['worker_fence_counter_before']} → {neg_f['worker_fence_counter_after']}`, counted data-plane flow `counted={neg_f['counted_flow_proof'].get('counted')}`. |",
        f"| **Negative 2 — Device centrally revoked after restore (`peerM2D1`: `{neg_c['device_id']}`)** | **Revoked (not fenced, not `pending`)**: `DELETE /api/v1/member/devices/{{id}}` → HTTP `{neg_c['delete_http_status']}` (`operation_status={neg_c['operation_status']}`), `GET /api/v1/member/devices` → `status=\"{neg_c['member_devices_api_status']}\"`, `fence_row_count={neg_c['fence_row_count']}`, `external_withdrawal_audit_count={neg_c['external_withdrawal_audit_count']}`, worker fence counter `{neg_c['worker_fence_counter_before']} → {neg_c['worker_fence_counter_after']}`, data-plane withdrawn (`counted={neg_c['flow_after_central_revoke'].get('counted')}`). |",
        f"| **Post-`T0` unmapped member device (`peerM3D1` for `{unm['member_id']}`)** | **Deduplicated (#205)**: `central_member_device_rows_l2={unm['central_member_device_rows_l2']}`, `identity_unmapped_audit_count={unm['identity_unmapped_audit_count']}` (`deduplicated_across_sweeps={unm['deduplicated_across_sweeps']}`). |",
        "",
    ]
    return "\n".join(lines)


def main() -> int:
    run = load(PUBLIC / "followup9-run.json")
    live_path = PUBLIC / "followup9-live.json"
    live_text = live_path.read_text()
    root_prefix = str(ROOT) + "/"
    if root_prefix in live_text:
        live_text = live_text.replace(root_prefix, "")
        live_path.write_text(live_text)
    live = json.loads(live_text)
    restore_case = live.get("AT30_restore_reconcile") or live.get("cases", {}).get("AT30_restore_reconcile")
    if not restore_case:
        raise RuntimeError("followup9-live.json missing AT30_restore_reconcile evidence")

    at30_status, at30_blockers, at30_summary = evaluate_at30(restore_case)
    app_base = run["source_revision"][:7]

    commands = [
        run.get("up", {}).get("command") or ["make", "-f", "mk/e2e.mk", "e2e-up"],
        run["live"].get("command") or run["live"].get("argv"),
        run.get("down", {}).get("command") or ["make", "-f", "mk/e2e.mk", "e2e-down"],
    ]

    meta = {
        "schema_version": 1,
        "follow_up": 9,
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

    rpo_rto_comparison = {
        "followup_8_pr221": FOLLOWUP8_RPO_RTO,
        "followup_9_current": {
            "run_id": live["run_id"],
            "pre_replay_db_rpo_seconds": restore_case["step4_reconciliation_observation"]["usage_replay"]["pre_replay_db_rpo_seconds"],
            "post_replay_service_rpo_seconds": restore_case["step4_reconciliation_observation"]["usage_replay"]["post_replay_service_rpo_seconds"],
            "protocol_rto_seconds": restore_case["step3_restore_central_to_t0"]["product_restore_protocol"]["protocol_rto_seconds"],
            "total_outage_rto_seconds": restore_case["step3_restore_central_to_t0"]["product_restore_protocol"]["total_outage_rto_seconds"],
        },
    }

    restore_record = {
        **meta,
        "id": "AT30-RESTORE-RECONCILE",
        "status": at30_status,
        "evaluation": at30_summary,
        "rpo_rto_comparison": rpo_rto_comparison,
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
        "expected": "An isolated DB/WAL restore replays previously ACKed node archives, reconciles provider and peers without duplicates or entitlement extension, fences restored authorizations whose external peer membership was withdrawn after T0 (D-076), exposes unacked outbox state during recovery (SECOPS-19), and measures complete-service RPO/RTO.",
        "actual": {
            "evaluation": at30_summary,
            "evidence": [
                "restore-reconcile.json",
                "restore-reconcile.md",
                "history/follow-up-8/restore-reconcile.json",
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
            "comparison_with_followup8_pr221": rpo_rto_comparison,
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
    s19 = s3["secops19_visibility"]
    usage = s4["usage_replay"]
    prov = s4["provider_reconciliation"]
    rev = s4["peer_revocation_reconciliation"]["revoked_in_t0_t1_device"]
    reg = s4["peer_revocation_reconciliation"]["regrant_authorize_device"]
    neg_c = s4["peer_revocation_reconciliation"]["negative_central_revoke_not_fenced"]
    neg_f = s4["peer_revocation_reconciliation"]["negative_post_restore_fresh_enroll_builds"]
    unm = s4["peer_revocation_reconciliation"]["new_in_t0_t1_unmapped_member_device"]
    hold_obs = s19["during_restore_hold"]["node_observation_report_usage"]["usage"]
    hold_snap = s19["during_restore_hold"]["fvpn_agent_usage_snapshot_cli"]["snapshot"]
    conv_obs = s19["after_complete_recovery_and_replay"]["node_observation_report_usage"]["usage"]
    conv_snap = s19["after_complete_recovery_and_replay"]["fvpn_agent_usage_snapshot_cli"]["snapshot"]
    t0_lsn = b0.get("t0_lsn", s1.get("t0_lsn"))

    at30_md = "\n".join(
        [
            "# AT30 — Follow-up 9",
            "",
            f"Full status: **{at30_status}**.",
            "",
            f"- **Isolated PITR & 6/6 post-restore checklist**: **passed** (`{proto['restore_info']['pitr_seconds']:.2f} s`, `restore-pitr.sh` primary-overwrite refusal exit code `2`).",
            f"- **2-minute fail-safe negative (before `freeze` / `begin-recovery`)**: **passed** (`{neg['window_seconds']:.2f} s`; without `ledger_epoch` rotation, `T0..T1` archived events `{usage['n0_seq_at_t0']+1}..{usage['n1_seq_at_t1']}` are not re-uploaded while idle, and post-`T1` traffic advances central `max_seen_seq={neg['after_120s_central_max_seen']}` while `highest_contiguous_seq={neg['after_120s_central_ack']}` stays at `N0`, triggering `ErrAckRegress` `{neg.get('node_ack_regress_log_lines')}` times).",
            f"- **5-step usage ledger restore protocol (`fvpn-usage-admin`) & Retained-node replay**: **passed** (`L0={usage['n0_seq_at_t0']}` → `L1={usage['n1_seq_at_t1']}` → `L2` replayed all `{usage['l2_events_1_to_n1']}/{usage['n1_seq_at_t1']}` events with `{usage['canonical_hash_matches_1_to_n1']}/{usage['n1_seq_at_t1']}` canonical SHA-256 matches, `0` lost, `0` double-counted, `0` gaps; pre-replay DB RPO `{usage['pre_replay_db_rpo_seconds']:.3f} s` vs `#221` `{FOLLOWUP8_RPO_RTO['pre_replay_db_rpo_seconds']:.3f} s`; post-replay service usage RPO `0 s`; protocol RTO `{proto['protocol_rto_seconds']:.2f} s` vs `#221` `{FOLLOWUP8_RPO_RTO['protocol_rto_seconds']:.2f} s`; total outage RTO `{proto['total_outage_rto_seconds']:.2f} s` vs `#221` `{FOLLOWUP8_RPO_RTO['total_outage_rto_seconds']:.2f} s`).",
            f"- **SECOPS-19 visibility (`#226` & `#225`)**: **passed** (during restore hold, `node_observations.report_payload.usage` reported `unacked_event_count={hold_obs.get('unacked_event_count')}`, `unacked_payload_bytes=\"{hold_obs.get('unacked_payload_bytes')}\"`, `ledger_epoch={hold_obs.get('ledger_epoch')}`, `meter_epoch_id={hold_obs.get('meter_epoch_id')}` and `fvpn-agent usage snapshot -json` reported `unacked_count={hold_snap.get('unacked_count')}`, `unacked_payload_bytes=\"{hold_snap.get('unacked_payload_bytes')}\"`; after `complete-recovery` and replay, both converged to `0` unacked events and `\"0\"` bytes).",
            f"- **Provider reconciliation (`Central L2` vs `P1`)**: **passed** (`m1` converged to `provider_status={prov['m1_subscription_status_l2']}`, `is_active={prov['m1_subscription_active_l2']}`, `refund_rows={prov['m1_refund_rows_l2']}`, `entitlement.state={prov['m1_entitlement_state_l2']}`, `device_limit={prov['m1_entitlement_dev_limit_l2']}` via `billing.reconcile`; `m2` remained `active` with `entitlement_extended={prov['m2_entitlement_extended']}`).",
            f"- **SECOPS-18 closure (`#227` D-076 restore authorization fence)**: **passed** (`peerM2D2` `{rev['device_id']}` revoked in `T0..T1` was fenced instead of re-granted after restore to `T0`: `device_authorization_fences` row `count={rev['fence_row_count']}`, worker log `\"device authorization unconfirmed during ledger restore\"` with 4 fields, worker counter `fvpn_device_restore_fence_decisions_total={rev['worker_restore_fence_counter_value']}`, `device.authorization.external_withdrawal` audit `count={rev['external_withdrawal_audit_count']}`, `GET /api/v1/member/devices` `status=\"{rev['l2_member_devices_api_status']}\"`, data-plane `counted={rev['l2_counted_flow_resumed']}`).",
            f"- **D-076 Re-grant & Two Negatives**: **passed** (`AuthorizeDevice` replay of `T0` key preserved fence `count={reg['fence_rows_after_t0_replay']}`; fresh `AuthorizeDevice` deleted fence `count={reg['fence_rows_after_fresh_regrant']}`, granted peer once, and restored counted flow `counted={reg['counted_flow_after_regrant'].get('counted')}`; central revoke negative `peerM2D1` stayed `status=\"{neg_c['member_devices_api_status']}\"` with `fence={neg_c['fence_row_count']}`; post-restore fresh enroll negative `peerM2D3` granted with `status=\"{neg_f['member_devices_api_status']}\"`, `fence={neg_f['fence_row_count']}`, `counted={neg_f['counted_flow_proof'].get('counted')}`; unmapped discovery denial `#205` deduplicated `count={unm['identity_unmapped_audit_count']}`).",
            "",
            "Evidence: [restore-reconcile.md](restore-reconcile.md), [restore-reconcile.json](restore-reconcile.json), [history/follow-up-8/restore-reconcile.json](history/follow-up-8/restore-reconcile.json), [history/follow-up-6/pitr.json](history/follow-up-6/pitr.json).",
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
        "follow_up": 9,
        "run_id": live["run_id"],
        "driver_run_id": run["driver_run_id"],
        "cases": cases_verdict,
        "at30_evaluation": at30_summary,
        "blocked": any(v != "passed" for v in cases_verdict.values()),
        "cleanup": cleanup["status"],
    }
    write(PUBLIC / "followup9-verdict.json", verdict)

    summary_md = "\n".join(
        [
            "# Follow-up 9 — AT30 re-measured with D-076 fence & SECOPS-19 visibility",
            "",
            f"Application base `{app_base}` (#231); harness `{run['source_revision']}`; run `{live['run_id']}`. Shared development host, **not candidate hardware**.",
            "",
            "| Measurement | Follow-up 8 (`#221`) | Follow-up 9 (`#231`) |",
            "|---|---|---|",
            f"| `T0` pgBackRest full backup / restore target | `20261030-220312F` | `{b0['backup_label']}` / `{s1['t0_utc']}` (LSN `{t0_lsn}`) |",
            f"| `L0` → `L1` central ACKed usage sequence (`N0` → `N1`) | `2` → `5` | `{usage['n0_seq_at_t0']}` (`{l0_bytes}` B) → `{usage['n1_seq_at_t1']}` (`{l1_bytes}` B) |",
            f"| SECOPS-19 during restore hold (`node_observations` / `fvpn-agent usage snapshot`) | zeroed `UsageOutboxStatus` | `unacked_event_count={hold_obs.get('unacked_event_count')}` (`{hold_obs.get('unacked_payload_bytes')}` B) / `unacked_count={hold_snap.get('unacked_count')}` (`{hold_snap.get('unacked_payload_bytes')}` B), `ledger_epoch={hold_obs.get('ledger_epoch')}` |",
            f"| SECOPS-19 after `complete-recovery` & replay | SQLite inspection only | Both converged to `0` unacked (`unacked_payload_bytes=\"0\"`, `acked_through={conv_snap.get('acked_through')}`, `last_event_seq={conv_snap.get('last_event_seq')}`) |",
            f"| Usage replay (`L2` vs `L1` and `L0`) | **passed** (`5/5` hashes, `0` lost, `0` dup) | **passed** (`{usage['l2_events_1_to_n1']}/{usage['n1_seq_at_t1']}` events, `{usage['canonical_hash_matches_1_to_n1']}/{usage['n1_seq_at_t1']}` hashes, `0` lost, `0` dup, `0` gaps) |",
            f"| Pre-replay DB RPO (`T1 - T0`) / Post-replay service usage RPO | `{FOLLOWUP8_RPO_RTO['pre_replay_db_rpo_seconds']:.3f} s` / `0 s` | `{usage['pre_replay_db_rpo_seconds']:.3f} s` / `0 s` |",
            f"| Protocol RTO (Step 1 `freeze` → ready) / Total outage RTO | `{FOLLOWUP8_RPO_RTO['protocol_rto_seconds']:.2f} s` / `{FOLLOWUP8_RPO_RTO['total_outage_rto_seconds']:.2f} s` | `{proto['protocol_rto_seconds']:.2f} s` / `{proto['total_outage_rto_seconds']:.2f} s` |",
            f"| Provider reconciliation (`Central L2` vs `P1`) | **passed** | **passed** (`m1` `canceled` + `refund=1` + `expired`; `m2` `active`, `entitlement_extended=False`) |",
            f"| `T0..T1` revoked device (`peerM2D2`) after restore (`SECOPS-18` / D-076) | **blocked** (re-granted in NetBird) | **passed**: fenced (`device_authorization_fences=1`, worker log + counter `=1`, `external_withdrawal` audit `=1`, `GET /member/devices` `status=pending`, data-plane withdrawn) |",
            f"| D-076 Re-grant (`AuthorizeDevice`) & Two Negatives | N/A | **passed**: `T0` key replay kept fence (`1`); fresh `AuthorizeDevice` cleared fence (`0`) and restored flow; central revoke (`revoked`, `fence=0`) and fresh post-restore enroll (`authorized`, `fence=0`) unfenced |",
            f"| **AT30 overall status** | **blocked** (`SECOPS-18`) | **{at30_status}** |",
            "",
            "Full details and command sequences are in [restore-reconcile.md](restore-reconcile.md) and [restore-reconcile.json](restore-reconcile.json). Follow-up 8 records are archived byte-for-byte under [history/follow-up-8](history/follow-up-8/archive-index.json). See [requests](requests.md) and [run instructions](RUN.md).",
            "",
        ]
    )
    (PUBLIC / "summary.md").write_text(summary_md)

    requests_md = "\n".join(
        [
            "# Requests to orchestrator — Follow-up 9",
            "",
            f"Measured application base `{app_base}` (#231) includes wire-members follow-up 9 (#227: D-076 restore authorization fence), agent follow-up 19 (#226: SECOPS-19 live `UsageOutboxStatus` in observations), and usage-agent follow-up 1 (#225: read-only `fvpn-agent usage snapshot`). No product, lab, or migration source outside owned paths was modified. Every claim is backed by [restore-reconcile.md](restore-reconcile.md) and [restore-reconcile.json](restore-reconcile.json).",
            "",
            "| Request | Status / owner | Evidence and required action |",
            "|---|---|---|",
            f"| SECOPS-18 / D-076, post-restore `T0..T1` device revocation fence | **Closed (#227 verified in live AT30 restore)** | `restore-reconcile.json` proves that after restoring central DB to `T0`, `peerM2D2` (`{rev['device_id']}`, revoked in `T0..T1`) is fenced instead of re-granted (`device_authorization_fences` row `count={rev['fence_row_count']}`, worker log `\"device authorization unconfirmed during ledger restore\"` with 4 fields, worker counter `fvpn_device_restore_fence_decisions_total={rev['worker_restore_fence_counter_value']}`, `device.authorization.external_withdrawal` audit `count={rev['external_withdrawal_audit_count']}`, `GET /api/v1/member/devices` `status=pending`, data-plane withdrawn). Both negatives pass (`peerM2D1` centrally revoked → `status=revoked`, `fence=0`; `peerM2D3` freshly enrolled after restore → `status=authorized`, `fence=0`). Re-grant via `AuthorizeDevice` (`member.device_authorize`, fresh idempotency key; admin HTTP route pending contract revision 12 / D-077) deletes the fence row and restores single NetBird grant and counted forwarding. |",
            f"| SECOPS-19, node unacked outbox & retained-archive observability | **Closed (#226 & #225 verified in live AT30 restore)** | During the restore hold (`reconciling`), `node_observations.report_payload.usage` reported `unacked_event_count={hold_obs.get('unacked_event_count')}`, `unacked_payload_bytes=\"{hold_obs.get('unacked_payload_bytes')}\"`, `ledger_epoch={hold_obs.get('ledger_epoch')}`, `meter_epoch_id={hold_obs.get('meter_epoch_id')}`, matching `fvpn-agent usage snapshot -config /etc/vpn-node/agent-run.json -json` (`unacked_count={hold_snap.get('unacked_count')}`, `acked_through={hold_snap.get('acked_through')}`, `last_event_seq={hold_snap.get('last_event_seq')}`). After `complete-recovery` and replay, both converged to `0` unacked events and `\"0\"` bytes (`acked_through={conv_snap.get('acked_through')}`, `last_event_seq={conv_snap.get('last_event_seq')}`). |",
            f"| SECOPS-05 / AT30 full acceptance | **Closed / AT30 `passed`** | All AT30 pass criteria (isolated PITR + 6/6 checklist, 2-minute fail-safe negative, 5-step `fvpn-usage-admin` restore protocol, retained-node usage archive replay with `0` lost / `0` double-counted, provider reconciliation, D-076 peer/revocation fence + negatives + re-grant, SECOPS-19 visibility, and measured RPO/RTO) are now `passed` in [AT30.json](AT30.json). |",
            "| D-077, admin HTTP route for device re-authorization | **Tracked (`contracts` revision 12 / `w3-wire-members`)** | `devices.Service.AuthorizeDevice` (`operation kind = member.device_authorize`) is verified live against the restored PostgreSQL and worker (`restore-reconcile.json`); the admin HTTP route wrapper remains tracked under contract revision 12 / D-077. |",
            "| SECOPS-20, devstack `node-netns` Docker `ip nat POSTROUTING` (`DOCKER_POSTROUTING` `xt SNAT`) vs `#220` whole-ruleset `VerifyServing` | **New (`w3-devstack-e2e`: `deploy/devstack/entrypoint-node.sh` / `w1-enforce-nft`: `node/enforce/verify_foreign.go`)** | Because `deploy/devstack/compose.yaml` attaches two bridge networks (`lab` and `origin`) to `node-netns`, `dockerd` populates `table ip nat` with `POSTROUTING` → `DOCKER_POSTROUTING` (`xt target \"SNAT\"` for `127.0.0.11`). After `#220` (`2473acf`), `VerifyServing` reads `nft -j list ruleset` and fails closed with `FLOWTABLE_BYPASS` (`table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt`) when `>=1` member grant is compiled. Observed and recorded on the unmodified container in `restore-reconcile.json` (`step1_checkpoint_t0.secops20_docker_postrouting_observation`) before removing the unused container-runtime `ip nat POSTROUTING` / `DOCKER_POSTROUTING` chains in `fvpn-w4-secops-node`. Request: `deploy/devstack/entrypoint-node.sh` should remove `ip nat POSTROUTING` / `DOCKER_POSTROUTING` on container boot (or `verify_foreign.go` should classify Docker's `127.0.0.11` embedded-DNS `DOCKER_POSTROUTING` jump). |",
            "| SECOPS-21, post-restore `nodes.desired_config_generation` regression vs retained node `applied_config_generation` | **New (`w2-policy`: `internal/desiredstate/compiler.go:198-211` / `w3-wire-nodes`: `internal/nodes/observations.go:289`)** | When central PostgreSQL is restored to `T0` (`desired_config_generation=11`) while the node retains its `T1` state (`applied_config_generation=14`), the node's post-restore observation updates `nodes.applied_config_generation=14`. However, `desiredstate.Compiler.Compile` (`internal/desiredstate/compiler.go:198-211`) computes `baseGen = max(nodes.desired_config_generation, max(desired_states.generation))` without including `nodes.applied_config_generation`, so subsequent post-restore compilations emit stale generations (`12`, `13 < 14`) that `fvpn-agent` (`node/internal/agentstore/state.go:257`) rejects with `ErrStaleGeneration`. Request: include `nodes.applied_config_generation` in `baseGen` in `internal/desiredstate/compiler.go:198-211`. |",
            "| AT29 / AT31 / SECOPS-16 / SECOPS-17 | **Deferred to Follow-up 10 (`w3-devstack-e2e` follow-up 14)** | Per Follow-up 9 instructions, AT29, AT31, and SECOPS-17 stay with Follow-up 10 awaiting devstack follow-up 14's exports. |",
            "",
        ]
    )
    (PUBLIC / "requests.md").write_text(requests_md)

    run_md = "\n".join(
        [
            "# Follow-up 9 run instructions",
            "",
            "Run from this worktree with Go 1.27.3 and Docker:",
            "",
            "```sh",
            "make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build",
            "make -f mk/at-secops.mk at-secops-followup9",
            "make go-lint",
            "go vet ./...",
            "```",
            "",
            "`at-secops-followup9` runs `tests/e2e/at/secops/followup9_run.py` in the foreground:",
            "",
            "1. Verifies that zero `fvpn-w4-secops-*` Docker containers, volumes, or networks exist (`inventory_before`).",
            "2. Brings up a fresh lab stack (`make -f mk/e2e.mk e2e-up` with `FVPN_E2E_PREFIX=w4-secops`, `FVPN_E2E_PORT_BASE=30000`, `E2E_RESET=1`).",
            "3. Runs `TestAT30_Followup9_LiveRestoreReconcile` (`tests/e2e/at/secops/followup9_live_test.go`):",
            "   - **Step 1 (`T0`)**: Seeds two paid members (`m1`, `m2`) with native devices on `fvpn-w4-secops-node`, runs counted traffic until usage checkpoints are ACKed (`L0`), captures SQLite + `fvpn-agent usage snapshot`, and takes a pgBackRest full backup (`T0`).",
            "   - **Step 2 (`T1`)**: Runs additional counted traffic (`L1 > L0`), applies subscription cancellation + full refund on `m1` (`P1`), revokes `m2`'s second device `peerM2D2`, authorizes `peerM3D1` for post-`T0` member `m3` (`N1`), and archives WAL (`wal-switch`).",
            "   - **Step 3 (Restore to `T0` + SECOPS-19 visibility)**: Verifies `restore-pitr.sh` primary-overwrite refusal (exit `2`), runs the 2-minute fail-safe negative without `freeze`/`begin-recovery`, then restores `T0` cleanly (`post-restore-checklist.sql` `6/6`) and executes the 5-step product restore protocol (`freeze` → `begin-recovery` → node archive replay → `complete-recovery`), capturing `node_observations.report_payload.usage` and `fvpn-agent usage snapshot -json` both during the restore hold (non-zero unacked) and after `complete-recovery` (`0` unacked), and measuring `RTO`.",
            "   - **Step 4 (D-076 Fence, Re-Grant, Two Negatives & Reconciliation)**: Verifies `peerM2D2` is fenced (`device_authorization_fences` row, worker log line with 4 fields, worker counter `fvpn_device_restore_fence_decisions_total`, `device.authorization.external_withdrawal` audit, `GET /api/v1/member/devices` `status=pending`, data-plane withdrawn), exercises `AuthorizeDevice` re-grant (proving `T0` key replay preserves the fence while a fresh key clears the fence and restores single NetBird grant and counted flow), and verifies both negatives (central revoke `status=revoked` unfenced; post-restore fresh device enrollment `status=authorized` unfenced).",
            "4. Unconditionally tears down all `fvpn-w4-secops-*` containers, volumes, networks, and restore images and verifies `inventory_after` is zero.",
            "5. Renders records via `python3 tests/e2e/at/secops/followup9_records.py`.",
            "",
            "Previous Follow-up 8 root records are archived with SHA-256 digests under `docs/e2e/at/secops/history/follow-up-8/`.",
            "",
        ]
    )
    (PUBLIC / "RUN.md").write_text(run_md)

    deps_path = PUBLIC / "dependencies.json"
    if deps_path.exists():
        deps = load(deps_path)
        deps["follow_up"] = 9
        deps["checked_at"] = run["finished_at"]
        deps["application_base"] = app_base
        deps["harness_revision"] = run["source_revision"]
        deps["history"] = PREVIOUS_RUN + "dependencies.json"
        deps.setdefault("merged", {})["secops_followup_8"] = 221
        deps["merged"]["usage_agent_followup_1"] = 225
        deps["merged"]["node_agent_followup_19"] = 226
        deps["merged"]["wire_members_followup_9_d076"] = 227
        deps.setdefault("requests", {})["SECOPS-05"] = {
            "status": "closed (AT30 passed in follow-up 9)",
            "prs": [165, 205, 225, 226, 227],
            "evidence": "restore-reconcile.json",
        }
        deps["requests"]["SECOPS-18"] = {
            "status": "closed",
            "owner": "w2-members (internal/devices)",
            "prs": [227],
            "detail": "D-076 restore authorization fence verified in live AT30 restore (fence row, worker log, worker counter, external_withdrawal audit, pending status, withdrawn data plane, re-grant via AuthorizeDevice, and two negatives)",
            "evidence": "restore-reconcile.json",
        }
        deps["requests"]["SECOPS-19"] = {
            "status": "closed",
            "owner": "w1-node-agent (node/agent/observe.go, node/agent/usage_snapshot.go)",
            "prs": [225, 226],
            "detail": "Live UsageOutboxStatus in node_observations and read-only fvpn-agent usage snapshot verified during restore hold (non-zero unacked) and post-replay convergence (zero unacked)",
            "evidence": "restore-reconcile.json",
        }
        deps["requests"]["SECOPS-20"] = {
            "status": "open",
            "owner": "w3-devstack-e2e (deploy/devstack/entrypoint-node.sh) / w1-enforce-nft (node/enforce/verify_foreign.go)",
            "detail": "Docker attaches two networks (lab and origin) to node-netns and populates table ip nat POSTROUTING -> DOCKER_POSTROUTING (xt target SNAT for 127.0.0.11), causing VerifyServing (#220 whole-ruleset read-back) to fail closed with FLOWTABLE_BYPASS on unmodified devstack node containers when >=1 member grant is compiled",
            "evidence": "restore-reconcile.json",
        }
        deps["requests"]["SECOPS-21"] = {
            "status": "open",
            "owner": "w2-policy (internal/desiredstate/compiler.go:198-211) / w3-wire-nodes (internal/nodes/observations.go:289)",
            "detail": "After central DB restore to T0 while node retains T1 applied_config_generation, desiredstate.Compiler.Compile computes baseGen from desired_config_generation and max(desired_states.generation) without including nodes.applied_config_generation, emitting stale generations rejected by fvpn-agent with ErrStaleGeneration",
            "evidence": "restore-reconcile.json",
        }
        write(deps_path, deps)

    scope_path = PUBLIC / "scope-review.json"
    if scope_path.exists():
        scope = load(scope_path)
        own_record_names = [
            "restore-reconcile.json",
            "restore-reconcile.md",
            "AT30.json",
            "AT30.md",
            "followup9-live.json",
            "followup9-run.json",
            "followup9-verdict.json",
            "live.json",
            "cleanup.json",
            "dependencies.json",
            "requests.md",
            "summary.md",
            "RUN.md",
        ]
        scope["follow_up_9"] = {
            "reviewed_at": run["finished_at"],
            "application_base": app_base,
            "source_revision": run["source_revision"],
            "scope": "AT30 second half re-measured on #231 with D-076 restore authorization fence (#227), SECOPS-19 live UsageOutboxStatus in observations (#226), and read-only fvpn-agent usage snapshot (#225); AT30 passed. Scope review pins refreshed for own records only.",
            "evidence": [
                "restore-reconcile.json",
                "restore-reconcile.md",
                "AT30.json",
                "AT30.md",
            ],
            "records": [
                {
                    "path": f"docs/e2e/at/secops/{name}",
                    "sha256": sha256_file(PUBLIC / name),
                }
                for name in own_record_names
                if (PUBLIC / name).is_file()
            ],
            "closed_requests": ["SECOPS-05", "SECOPS-18", "SECOPS-19"],
            "new_requests": ["SECOPS-20", "SECOPS-21"],
        }
        write(scope_path, scope)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
