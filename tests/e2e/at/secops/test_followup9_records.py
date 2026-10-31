#!/usr/bin/env python3
"""Unit tests for Follow-up 9 AT30 restore, D-076 fence & SECOPS-19 visibility evaluation."""
import copy
import unittest

from followup9_records import clean_inventory, evaluate_at30


def sample_restore_case() -> dict:
    return {
        "step1_checkpoint_t0": {"t0_utc": "2026-10-30T23:10:00Z"},
        "step2_world_forward_t1": {"t1_utc": "2026-10-30T23:12:00Z"},
        "step3_restore_central_to_t0": {
            "primary_overwrite_refusal": {"exit_code": 2, "refused": True},
            "fail_safe_negative_without_protocol": {
                "window_seconds": 120.5,
                "retained_t0_t1_skipped_without_protocol": True,
                "node_ack_regress_log_lines": 3,
            },
            "product_restore_protocol": {
                "restore_info": {"checklist_sections": 6, "pitr_seconds": 8.4},
                "mode_after_freeze": "frozen",
                "mode_after_begin_recovery": "reconciling",
                "mode_after_complete_recovery": "normal",
                "reconciling_rollups_remaining": 0,
                "member_api_ready": {"m2_me_status": 200, "m2_usage_status": 200},
                "node_observation_accepted_after_recovery": True,
                "protocol_rto_seconds": 102.4,
                "total_outage_rto_seconds": 108.1,
            },
            "secops19_visibility": {
                "during_restore_hold": {
                    "node_observation_report_usage": {
                        "usage": {
                            "unacked_event_count": 4,
                            "unacked_payload_bytes": "1680",
                            "ledger_epoch": "01a18f00-0000-7000-8000-000000000001",
                            "meter_epoch_id": "01a18f00-0000-7000-8000-000000000002",
                        }
                    },
                    "fvpn_agent_usage_snapshot_cli": {
                        "snapshot": {
                            "unacked_count": 4,
                            "unacked_payload_bytes": "1680",
                            "ledger_epoch": "01a18f00-0000-7000-8000-000000000001",
                            "meter_epochs": ["01a18f00-0000-7000-8000-000000000002"],
                        }
                    },
                },
                "after_complete_recovery_and_replay": {
                    "node_observation_report_usage": {
                        "usage": {
                            "unacked_event_count": 0,
                            "unacked_payload_bytes": "0",
                            "ledger_epoch": "01a18f00-0000-7000-8000-000000000001",
                            "meter_epoch_id": "01a18f00-0000-7000-8000-000000000002",
                        }
                    },
                    "fvpn_agent_usage_snapshot_cli": {
                        "snapshot": {
                            "unacked_count": 0,
                            "unacked_payload_bytes": "0",
                            "ledger_epoch": "01a18f00-0000-7000-8000-000000000001",
                            "meter_epochs": ["01a18f00-0000-7000-8000-000000000002"],
                        }
                    },
                },
            },
        },
        "step4_reconciliation_observation": {
            "usage_replay": {
                "status": "passed",
                "n0_seq_at_t0": 2,
                "n1_seq_at_t1": 5,
                "l2_events_1_to_n1": 5,
                "canonical_hash_matches_1_to_n1": 5,
                "lost_t0_t1_acked_events": 0,
                "double_counted_events": 0,
                "usage_ledger_gaps": 0,
                "pre_replay_db_rpo_seconds": 115.0,
                "post_replay_service_rpo_seconds": 0,
            },
            "provider_reconciliation": {
                "status": "passed",
                "m1_subscription_status_l2": "canceled",
                "m1_subscription_active_l2": False,
                "m1_refund_rows_l2": 1,
                "m2_subscription_status_l2": "active",
                "m2_subscription_active_l2": True,
                "m2_entitlement_extended": False,
            },
            "peer_revocation_reconciliation": {
                "revoked_in_t0_t1_device": {
                    "device_id": "01a18f00-0000-7000-8000-000000000010",
                    "external_peer_id": "peer-m2-d2",
                    "fenced_instead_of_regranted": True,
                    "fence_row_count": 1,
                    "external_withdrawal_audit_count": 1,
                    "worker_restore_fence_counter_value": 1,
                    "worker_restore_fence_log_lines": [
                        {
                            "msg": "device authorization unconfirmed during ledger restore",
                            "device_id": "01a18f00-0000-7000-8000-000000000010",
                            "member_id": "01a18f00-0000-7000-8000-000000000020",
                            "peer_id": "peer-m2-d2",
                            "ledger_mode": "reconciling",
                        }
                    ],
                    "l2_member_devices_api_status": "pending",
                    "l2_observed_authorized": False,
                    "l2_readded_to_netbird_paid_group": False,
                    "l2_counted_flow_resumed": False,
                },
                "regrant_authorize_device": {
                    "passed": True,
                    "t0_idempotency_replay_replayed": True,
                    "fence_rows_after_t0_replay": 1,
                    "fence_rows_after_fresh_regrant": 0,
                    "observed_authorized_after_sync": True,
                    "member_devices_api_status": "authorized",
                    "counted_flow_after_regrant": {"counted": True},
                },
                "negative_central_revoke_not_fenced": {
                    "passed": True,
                    "member_devices_api_status": "revoked",
                    "fence_row_count": 0,
                    "external_withdrawal_audit_count": 0,
                    "worker_fence_counter_before": 1,
                    "worker_fence_counter_after": 1,
                },
                "negative_post_restore_fresh_enroll_builds": {
                    "passed": True,
                    "member_devices_api_status": "authorized",
                    "fence_row_count": 0,
                    "external_withdrawal_audit_count": 0,
                    "worker_fence_counter_before": 1,
                    "worker_fence_counter_after": 1,
                    "counted_flow_proof": {"counted": True},
                },
                "new_in_t0_t1_unmapped_member_device": {
                    "central_member_device_rows_l2": 0,
                    "identity_unmapped_audit_count": 1,
                    "deduplicated_across_sweeps": True,
                },
            },
        },
    }


class TestFollowup9Records(unittest.TestCase):
    def test_passes_when_d076_fence_regrant_negatives_and_secops19_pass(self):
        status, blockers, summary = evaluate_at30(sample_restore_case())
        self.assertEqual(status, "passed")
        self.assertEqual(blockers, [])
        self.assertTrue(all(summary.values()))

    def test_blocks_when_d076_fence_missing_log_or_counter(self):
        case = copy.deepcopy(sample_restore_case())
        case["step4_reconciliation_observation"]["peer_revocation_reconciliation"][
            "revoked_in_t0_t1_device"
        ]["worker_restore_fence_counter_value"] = 0
        status, blockers, summary = evaluate_at30(case)
        self.assertEqual(status, "blocked")
        self.assertFalse(summary["t0_t1_device_revocation_fenced_d076"])
        self.assertTrue(any("D-076 fence check failed" in b for b in blockers))

    def test_blocks_when_regrant_leaves_fence_or_fails_flow(self):
        case = copy.deepcopy(sample_restore_case())
        case["step4_reconciliation_observation"]["peer_revocation_reconciliation"][
            "regrant_authorize_device"
        ]["fence_rows_after_fresh_regrant"] = 1
        status, blockers, summary = evaluate_at30(case)
        self.assertEqual(status, "blocked")
        self.assertFalse(summary["regrant_authorize_device_clears_fence"])

    def test_blocks_when_secops19_hold_is_zero_or_fails_to_converge(self):
        case = copy.deepcopy(sample_restore_case())
        case["step3_restore_central_to_t0"]["secops19_visibility"][
            "during_restore_hold"
        ]["node_observation_report_usage"]["usage"]["unacked_event_count"] = 0
        status, blockers, summary = evaluate_at30(case)
        self.assertEqual(status, "blocked")
        self.assertFalse(summary["secops19_visibility_and_zero_convergence"])

    def test_clean_inventory_check(self):
        self.assertTrue(
            clean_inventory(
                {
                    "containers": {"exit_code": 0, "count": 0},
                    "volumes": {"exit_code": 0, "count": 0},
                    "networks": {"exit_code": 0, "count": 0},
                }
            )
        )


if __name__ == "__main__":
    unittest.main()
