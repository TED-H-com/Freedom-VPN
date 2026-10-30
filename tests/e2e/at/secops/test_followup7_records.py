"""Prevent missing fault/lifecycle evidence from becoming a passing record."""
import copy
import unittest
from followup7_records import cutoff_verdict, protected_verdict


class VerdictTests(unittest.TestCase):
    def test_protected_credentials_need_deletion_proof(self):
        pre=[{'present':True,'mode':'0:600','device_flow_keys':1} for _ in range(60)]
        post=[{'scan_ok':True,'deletion_proven':True,'post_join_present':False} for _ in range(60)]
        self.assertTrue(protected_verdict(pre,post))  # D-066 permits this protected class.
        post[0]['deletion_proven']=False
        self.assertFalse(protected_verdict(pre,post))
        post[0]['deletion_proven']=True
        pre[0]['mode']='0:644'
        self.assertFalse(protected_verdict(pre,post))
        self.assertFalse(protected_verdict(pre[1:],post[1:]))

    def test_cutoff_requires_positive_control_and_recovery(self):
        row={'before_fault':{'counted':True},'signed_node_duration_seconds':60,
             'worker_pause_ok':True,'control_stop_ok':True,'agent_kill_command_ok':True,
             'samples':[{'request_start_after_deadline':True,'http_200':False}],
             'recovery_flow':{'counted':True}}
        self.assertTrue(cutoff_verdict(row))
        for mutation in ({'samples':[]},{'before_fault':{'counted':False}},
                         {'recovery_flow':{'counted':False}}, {'signed_node_duration_seconds':86400},
                         {'samples':[{'request_start_after_deadline':True,'http_200':True}]}):
            changed=copy.deepcopy(row);changed.update(mutation)
            self.assertFalse(cutoff_verdict(changed))


if __name__=='__main__':unittest.main()
