#!/usr/bin/env python3
"""Unit tests for Follow-up 8 AT30 restore & reconciliation record evaluation."""
import copy
import unittest

from followup8_records import clean_inventory, evaluate_at30


def sample_restore_case(revocation_preserved: bool = False) -> dict:
    return {
        "step1_checkpoint_t0": {"t0_utc": "2026-10-30T21:10:00Z"},
        "step2_world_forward_t1": {"t1_utc": "2026-10-30T21:12:00Z"},
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
                "protocol_rto_seconds": 14.2,
                "total_outage_rto_seconds": 26.8,
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
                "pre_replay_db_rpo_seconds": 120.0,
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
                    "l2_central_status_after_reconcile": (
                        "revoked" if revocation_preserved else "authorized"
                    ),
                    "l2_readded_to_netbird_paid_group": (not revocation_preserved),
                    "l2_counted_flow_resumed": (not revocation_preserved),
                },
                "new_in_t0_t1_unmapped_member_device": {
                    "central_member_device_rows_l2": 0,
                    "identity_unmapped_audit_count": 1,
                    "deduplicated_across_sweeps": True,
                },
                "new_in_t0_t1_existing_member_device": {
                    "rediscovered_rows_in_l2": 1,
                    "rediscovered_status_in_l2": "authorized",
                },
            },
        },
    }


class TestFollowup8Records(unittest.TestCase):
    def test_blocks_when_revoked_device_reauthorized_after_restore(self):
        status, blockers, summary = evaluate_at30(
            sample_restore_case(revocation_preserved=False)
        )
        self.assertEqual(status, "blocked")
        self.assertTrue(summary["retained_node_usage_replay"])
        self.assertTrue(summary["provider_reconciliation"])
        self.assertTrue(summary["unmapped_peer_discovery_deduplicated"])
        self.assertFalse(summary["t0_t1_device_revocation_preserved"])
        self.assertEqual(len(blockers), 1)
        self.assertIn("SECOPS-18", blockers[0])

    def test_passes_when_all_subsystems_and_revocation_converge(self):
        status, blockers, summary = evaluate_at30(
            sample_restore_case(revocation_preserved=True)
        )
        self.assertEqual(status, "passed")
        self.assertEqual(blockers, [])
        self.assertTrue(all(summary.values()))

    def test_blocks_when_usage_replay_loses_events(self):
        case = copy.deepcopy(sample_restore_case(revocation_preserved=True))
        case["step4_reconciliation_observation"]["usage_replay"]["lost_t0_t1_acked_events"] = 1
        status, blockers, summary = evaluate_at30(case)
        self.assertEqual(status, "blocked")
        self.assertFalse(summary["retained_node_usage_replay"])
        self.assertTrue(any("Retained-node usage archive replay" in b for b in blockers))

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
        self.assertFalse(
            clean_inventory(
                {
                    "containers": {"exit_code": 0, "count": 1},
                    "volumes": {"exit_code": 0, "count": 0},
                }
            )
        )


if __name__ == "__main__":
    unittest.main()
