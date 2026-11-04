import unittest
import sys
sys.dont_write_bytecode = True
from admission import suite_roots, process_args


class AdmissionTest(unittest.TestCase):
    def test_unmatched_quote_is_retained_as_a_candidate(self):
        args = process_args('python3 gates.sh --label "unfinished')
        self.assertEqual(args[:2], ['python3', 'gates.sh'])
        self.assertEqual(args[-1], '"unfinished')

    def test_orchestration_shell_is_still_recognizable(self):
        args = process_args("bash -c echo 'mk/at-members.mk")
        self.assertEqual(args[:2], ['bash', '-c'])
        self.assertEqual(process_args('make -f mk/at-members.mk at-members'),
                         ['make', '-f', 'mk/at-members.mk', 'at-members'])

    def test_nested_make_and_current_runner_do_not_double_count(self):
        parents = {10: 1, 11: 10, 20: 1, 21: 20, 30: 1, 31: 30}
        self.assertEqual(suite_roots({10, 11, 20, 21, 30}, parents, 31), [10, 20])

    def test_two_standalone_targets_stop_admission(self):
        self.assertEqual(len(suite_roots({10, 20}, {10: 1, 20: 1}, 30)), 2)


if __name__ == '__main__':
    unittest.main()
