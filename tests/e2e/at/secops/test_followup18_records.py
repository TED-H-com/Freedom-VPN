import unittest
from followup18_records import audit


class Followup18Records(unittest.TestCase):
    def test_no_resolution_without_fault_cutoff(self):
        self.assertFalse(audit({},[])['answered'])

    def test_passed_case_without_section_fails(self):
        result=audit(dict(transport_cases=dict(lease_expiry=dict(status='passed'))),[])
        self.assertFalse(result['passed'])
        self.assertFalse(result['answered'])

    def test_cutoff_without_local_read_fails(self):
        result=audit(dict(transport_cases=dict(lease_expiry=dict(status='failed',cutoff_samples=[{},{}]))),[])
        self.assertFalse(result['passed'])
