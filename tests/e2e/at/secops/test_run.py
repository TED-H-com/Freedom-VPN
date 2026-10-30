import json
import unittest
from unittest.mock import patch
from run import scan, summarize_go, write_followup3_records, write_followup4_records


class EvidenceTests(unittest.TestCase):
    def events(self, *events):
        return '\n'.join(json.dumps(e) for e in events).encode()

    def test_missing_or_skipped_required_probe_cannot_pass(self):
        package = {'Action': 'pass', 'Package': 'example'}
        self.assertEqual(summarize_go(self.events(package), ['TestAT28'], 0)['status'], 'blocked')
        skipped = {'Action': 'skip', 'Package': 'example', 'Test': 'TestAT28'}
        self.assertEqual(summarize_go(self.events(skipped, package), ['TestAT28'], 0)['status'], 'blocked')
        passed = dict(skipped, Action='pass')
        self.assertEqual(summarize_go(self.events(passed, package), ['TestAT28'], 0)['status'], 'passed')
        self.assertEqual(summarize_go(self.events(passed, package), ['TestAT28'], 1)['status'], 'failed')

    def test_subtest_failure_and_skip_are_not_hidden_by_parent(self):
        events = [{'Action': 'pass', 'Package': 'example'},
                  {'Action': 'pass', 'Package': 'example', 'Test': 'TestAT28'},
                  {'Action': 'fail', 'Package': 'example', 'Test': 'TestAT28/expired'}]
        self.assertEqual(summarize_go(self.events(*events), ['TestAT28'], 0)['status'], 'failed')
        events[-1]['Action'] = 'skip'
        self.assertEqual(summarize_go(self.events(*events), ['TestAT28'], 0)['status'], 'blocked')

    def test_scan_reports_locations_without_secret_values(self):
        # TEST ONLY - NOT A SECRET; assembled at runtime and never stored.
        secret = b'fvpn_enr_' + b'x' * 32
        result = scan(b'clean\n' + secret + b'\n')
        self.assertEqual(result, [{'category': 'enrollment_token', 'line': 2}])
        self.assertNotIn(secret.decode(), json.dumps(result))
        self.assertEqual(scan(b'pass TestAT28\n'), [])
        pem = b'-----BEGIN ' + b'EC PRIVATE KEY-----'
        self.assertEqual(scan(pem)[0]['category'], 'private_key')
        dsn = b'postgresql://' + b'user:TEST_ONLY@example.invalid/db'
        self.assertEqual(scan(dsn)[0]['category'], 'credential_url')

    def test_non_json_diagnostics_cannot_create_pass(self):
        self.assertEqual(summarize_go(b'PASS\n', ['TestAT28'], 0)['status'], 'blocked')

    def test_early_publication_refusal_cannot_pass_later_worker_gate(self):
        live = {'publication_request': {'status': 422, 'code': 'NODE_NOT_SERVICEABLE'},
                'agent_node_incarnation': 'TEST ONLY agent incarnation'}
        with patch('run.write_json') as writer:
            write_followup3_records({'stack': {}, 'node_incarnation': 'TEST ONLY other node'}, live)
        records = {call.args[1]['id']: call.args[1] for call in writer.call_args_list}
        publication = records['SECOPS-11']
        self.assertEqual(publication['negative_status'], 'blocked')
        self.assertEqual(publication['node_incarnation'], live['agent_node_incarnation'])
        self.assertIsNone(publication['actual']['publication_operation'])
        self.assertFalse(publication['actual']['publication_negative']['passed'])
        self.assertEqual(records['SECOPS-06']['status'], 'blocked')


    def test_followup4_requires_live_probe_refusal_and_zero_binding_positive(self):
        step = {'name':'data_path_probe','status':'failed','message':'member_forwarding_unconfirmed','error_code':'NODE_NOT_SERVICEABLE'}
        live = {'publication_without_capacity': {'publication_request':{'status':202}, 'publication_operation':{'status':'running','steps':[step]}},
                'publication_empty_bindings': {'publication_request':{'status':202},'publication_operation':{'status':'succeeded'},'signed_device_bindings':0,'signed_publication_intent':'published'}}
        with patch('run.write_json') as writer:
            write_followup4_records({'stack':{}},live)
        publication = writer.call_args_list[-1].args[1]
        self.assertEqual(publication['status'],'passed')
        self.assertIn('does not assert a terminal',publication['negative_scope'])
        live['publication_without_capacity']['publication_request']['status'] = 422
        live['publication_empty_bindings']['signed_device_bindings'] = 1
        with patch('run.write_json') as writer:
            write_followup4_records({'stack':{}},live)
        publication = writer.call_args_list[-1].args[1]
        self.assertEqual(publication['negative_status'],'blocked')
        self.assertEqual(publication['positive_status'],'blocked')


if __name__ == '__main__':
    unittest.main()

class Followup5EvidenceTests(unittest.TestCase):
    def test_flow_requires_all_ordered_gates_not_a_claimed_flag(self):
        from followup5_records import counted_proof
        self.assertFalse(counted_proof({'counted':True,'http_200':True}))
        proof={'route_device':'fvnb0','http_200':True,'origin_authenticated':True,'origin_probe_matches':True,'origin_node_snat':True,'counter_reads_ok':True,'admitted_bytes_delta':{'upload':'1','download':'1'}}
        self.assertTrue(counted_proof(proof))
        proof['origin_node_snat']=False
        self.assertFalse(counted_proof(proof))

    def test_body_boundary_requires_the_complete_triplet(self):
        from followup5_records import body_verdict
        rows=[{'bytes':n,'passed':True} for n in (7,8,9)]
        self.assertEqual(body_verdict(rows,8),'passed')
        self.assertEqual(body_verdict(rows[:2],8),'blocked')
        rows[-1]['passed']=False
        self.assertEqual(body_verdict(rows,8),'failed')
