"""Consumer regression tests: exact joins, missing evidence and escalation counts."""
import copy
import unittest
from review_holds import joins


def fixture():
    at = '2026-11-01T12:00:00Z'
    report = dict(report_id='report', node_id='node', node_incarnation_id='incarnation', boot_id='boot',
        lease=dict(lease_id='lease'), applied_config_generation='30', observed_at=at,
        forwarding_block_reasons=['policy_missing'], member_forwarding_enabled=False,
        checks=[], errors=[dict(stage='lease', code='STALE_GENERATION')])
    sample = dict(report_id='report', node_id='node', incarnation_id='incarnation', boot_id='boot',
        lease_id='lease', applied_generation='30', lease_generation='29', hold_id='incident',
        reported_steps='1', reported_elapsed_ms='0', reported_alarm=False, summary_valid=True)
    incident = dict(id='incident', first_seen_at=at, last_seen_at=at, escalated_at=None, ended_at=None, end_reason=None)
    lease = dict(signed=dict(lease_id='lease', node_id='node', node_incarnation_id='incarnation',config_generation='29'), signatures=[dict(test_only=True)])
    row = dict(report=report, observation_id='observation', received_at=at, desired_generation_at_read='30', lease=lease,
        stored_lease_generation='29', classification_available=True, reverse_array_replay_available=True,
        publication_data_path_failure='stale_generation', reverse_array_replay_cause='stale_generation', operations_at_read=[])
    return dict(reports=[row], final=dict(incidents=[incident], samples=[sample], audits=[]))


class HoldEvidenceTests(unittest.TestCase):
    def test_M3_exact_join_with_equal_ack_and_desired(self):
        result = joins(fixture())
        self.assertEqual(result['problems'], [])
        self.assertTrue(result['holds'][0]['applied_equals_desired_at_read'])

    def test_M3_missing_or_foreign_evidence_never_qualifies(self):
        for mutate in (
            lambda v:v['final'].update(samples=[]),
            lambda v:v['reports'][0]['lease']['signed'].update(node_incarnation_id='foreign'),
            lambda v:v['final']['samples'][0].update(lease_id='other'),
            lambda v:v['reports'][0].update(reverse_array_replay_cause='forwarding_disabled'),
            lambda v:v['reports'][0]['report'].update(member_forwarding_enabled=True),
        ):
            v=fixture();mutate(v)
            self.assertTrue(joins(v)['problems'])

    def test_M3_unrelated_failure_keeps_incident_and_wins_both_orders(self):
        v=fixture();row=v['reports'][0]
        row['report']['errors'].append(dict(stage='apply_candidate',code='TC_FOREIGN_FILTER'))
        self.assertTrue(joins(v)['problems'])
        row.update(publication_data_path_failure='check_failed',reverse_array_replay_cause='check_failed')
        result=joins(v)
        self.assertEqual(result['problems'],[])
        self.assertEqual(result['holds'][0]['incident']['id'],'incident')

    def test_M3_age_escalation_requires_30_seconds_and_exactly_one_event(self):
        v=fixture();v['final']['incidents'][0]['escalated_at']='2026-11-01T12:00:31Z'
        event=dict(id='audit',occurred_at='2026-11-01T12:00:31Z',evidence=dict(incident_id='incident',alarm_report_id=None))
        v['final']['audits']=[event]
        result=joins(v)
        self.assertEqual(result['problems'],[])
        self.assertEqual(result['escalations'][0]['trigger'],'central_age_30s')
        bad=copy.deepcopy(v);bad['final']['audits'].append(event)
        self.assertTrue(joins(bad)['problems'])
        event['occurred_at']='2026-11-01T12:00:29Z'
        self.assertTrue(joins(v)['problems'])

    def test_M3_end_requires_exact_matching_receipt(self):
        v=fixture();v['final']['incidents'][0].update(ended_at='2026-11-01T12:00:02Z',end_reason='generation_matched')
        self.assertTrue(joins(v)['problems'])
        row=copy.deepcopy(v['reports'][0]);row['received_at']='2026-11-01T12:00:02Z'
        row['report'].update(report_id='matching',errors=[],forwarding_block_reasons=[],applied_config_generation='29')
        v['reports'].append(row)
        result=joins(v)
        self.assertEqual(result['problems'],[])
        self.assertEqual(result['matching_ends'][0]['matching_report_ids'],['matching'])


if __name__ == '__main__':
    unittest.main()
