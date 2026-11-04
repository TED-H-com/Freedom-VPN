import json
import os
from pathlib import Path
import tempfile
import types
import unittest
from unittest.mock import patch

import timing_admission as timing


class TimingAdmission18(unittest.TestCase):
    def test_held_before_start_delays_section(self):
        with tempfile.TemporaryDirectory() as tmp:
            journal, active = str(Path(tmp)/'journal'), str(Path(tmp)/'active')
            env = dict(FVPN_SECOPS_TIMING_JOURNAL=journal, FVPN_SECOPS_TIMING_ACTIVE=active, FVPN_HOLD_SH='TEST-ONLY')
            clock = [100]
            def stamp():
                clock[0] += 1
                return dict(at='TEST ONLY', mono_ns=str(clock[0]))
            def sleep(n):
                self.assertEqual(n,60)
                self.assertFalse(Path(active).exists())
                clock[0] += n*1_000_000_000
            responses = [types.SimpleNamespace(returncode=1,stdout='held'),types.SimpleNamespace(returncode=0,stdout='free')]
            original = timing.admit
            with patch.dict(os.environ,env),patch.object(timing,'stamp',side_effect=stamp),patch.object(timing.subprocess,'run',side_effect=responses) as run,patch.object(timing,'admit',side_effect=lambda command: original(command,sleep=sleep)):
                timing.begin('lease_expiry')
                clock[0]+=1
                timing.before_command('docker')
                self.assertEqual(run.call_count,2)
                with self.assertRaisesRegex(RuntimeError,'forbidden'):
                    timing.check('docker')
                clock[0]+=1
                timing.end('lease_expiry')
            events=[json.loads(line) for line in Path(journal).read_text().splitlines()]
            result=timing.validate(events)
            self.assertTrue(result['passed'],result)
            self.assertEqual(events[0]['answer'],'held')
            self.assertGreaterEqual(int(result['sections'][0]['start']['mono_ns'])-int(events[0]['mono_ns']),60_000_000_000)

    def test_held_inside_fails_evidence(self):
        events=[dict(kind='section_start',section='grant_expiry',mono_ns='100'),
                dict(kind='admission',command='docker',answer='held',mono_ns='110'),
                dict(kind='section_end',section='grant_expiry',mono_ns='120')]
        result=timing.validate(events)
        self.assertFalse(result['passed'])
        self.assertIn('held admission inside grant_expiry',result['errors'])
        events[1]['answer']='free'
        self.assertFalse(timing.validate(events)['passed'])

    def test_unclosed_section_fails(self):
        self.assertFalse(timing.validate([dict(kind='section_start',section='lease_expiry',mono_ns='1')])['passed'])
