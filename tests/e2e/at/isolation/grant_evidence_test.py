"""Receipt and original-wire negative checks against retained assembled inputs."""
import copy
import json
from pathlib import Path
import unittest
from grant_evidence import validate


class GrantEvidenceTests(unittest.TestCase):
    def test_D087_accepted_receipts_and_failure_boundaries(self):
        root=Path(__file__).resolve().parents[4]/'docs/e2e/at/isolation'
        transition=json.loads((root/'member-grants-transition.json').read_text())
        events=json.loads((root/'member-grants-wire.json').read_text())
        self.assertEqual(validate(transition,events)['status'],'pass')
        for failure in ('not_accepted','expired','wrong_lease','generation','early_true','counter','invariants','wire_digest'):
            with self.subTest(failure=failure):
                t,e=copy.deepcopy(transition),copy.deepcopy(events)
                if failure=='not_accepted':t['provisional_receipt']['http_status']=429
                if failure=='expired':t['provisional_receipt']['request']['forwarding_block_reasons']=['lease_expired']
                if failure=='wrong_lease':t['empty_lease']['response']['signed']['lease_id']='different'
                if failure=='generation':t['serving_receipt']['request']['applied_config_generation']='0'
                if failure=='early_true':t['true_flip']['started']=t['false_flip']['completed']
                if failure=='counter':t['counted_forwarding']['admission_bytes_after']=t['counted_forwarding']['admission_bytes_before']
                if failure=='invariants':t['after']['frozen']=True
                if failure=='wire_digest':next(v for v in e if v.get('request_wire'))['request_sha256']='0'*64
                with self.assertRaises(AssertionError):validate(t,e)


if __name__=='__main__':
    unittest.main()
