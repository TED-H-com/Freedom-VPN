"""Missing evidence cannot close restored generations or protected streams."""
import copy
import unittest
from followup10_records import generation_verdict, protected_verdict
from followup9_records import evaluate_at30
from test_followup9_records import sample_restore_case


class Followup10VerdictTests(unittest.TestCase):
    def test_generation_requires_first_offer_applied_not_just_clean_logs(self):
        row = {'offers':[{'generation':'15'}], 'observations':[{'applied':'15'}],
               'retained_at_t1':{'applied':'14'}, 'manual_generation_adjustment':False,
               'stale_generation_log_count':0}
        self.assertTrue(generation_verdict(row))
        for mutation in ({'offers':[]}, {'offers':[{'generation':'13'},{'generation':'15'}]},
                         {'observations':[{'applied':'16'}]}, {'manual_generation_adjustment':True},
                         {'retained_at_t1':{}}, {'stale_generation_log_count':1}):
            self.assertFalse(generation_verdict(dict(row, **mutation)))

    def test_final_bulk_attribution_needs_every_pre_and_post_delete(self):
        streams = [{'stream':f'peer-{n}', 'result':'scanned', 'deleted':True,
                    'scan':{'present':True, 'mode':'0:600', 'device_flow_keys':1}} for n in range(60)]
        rows = [{'pre_delete':s, 'post_delete_scan_ok':True, 'post_delete_present':False} for s in streams]
        bulk = {'counts':{'streams':streams}, 'error':False}
        self.assertTrue(protected_verdict(rows, bulk))
        self.assertFalse(protected_verdict(rows[:-1], bulk))
        for location, key, value in [('pre_delete','deleted',False), ('scan','mode','0:644'),
                                     ('post','post_delete_present',True), ('post','post_delete_scan_ok',False)]:
            bad = copy.deepcopy(rows)
            target = bad[0] if location == 'post' else bad[0]['pre_delete']
            if location == 'scan': target = target['scan']
            target[key] = value
            self.assertFalse(protected_verdict(bad, bulk))

    def test_old_service_method_cannot_satisfy_http_regrant(self):
        case = sample_restore_case()
        status, _, _ = evaluate_at30(case, http_regrant=True)
        self.assertEqual(status, 'blocked')
        peers = case['step4_reconciliation_observation']['peer_revocation_reconciliation']
        reg = peers['regrant_authorize_device']
        reg.update(existing_operation_returned=True, operation_kind='member.device_authorize',
                   same_key_replay_same_operation=True, replay_resource_version_unchanged=True,
                   replay_outbox_count_unchanged=True)
        peers['negative_central_revoke_not_fenced']['admin_authorize_revoked'] = {'status':409,'error':'STATE_CONFLICT'}
        self.assertEqual(evaluate_at30(case, http_regrant=True)[0], 'passed')
        reg['replay_outbox_count_unchanged'] = False
        self.assertEqual(evaluate_at30(case, http_regrant=True)[0], 'blocked')
