"""TEST ONLY synthetic process tables; no Docker or host mutation."""
import unittest
import followup39_timing as runner


class QualificationTests(unittest.TestCase):
    def test_stat_command_can_contain_spaces_and_parentheses(self):
        fields = ['S', '44'] + ['0'] * 9 + ['250', '150'] + ['0'] * 6 + ['900']
        got = runner.parse_stat('123 (a process (with spaces)) ' + ' '.join(fields))
        self.assertEqual(got, {'comm': 'a process (with spaces)', 'ppid': 44, 'ticks': 400, 'start': 900})

    def test_descendants_survive_reparenting_but_pid_reuse_does_not(self):
        remembered = set()
        first = {10: {'ppid': 1, 'start': 10}, 11: {'ppid': 10, 'start': 11}, 12: {'ppid': 11, 'start': 12}}
        self.assertEqual(runner.descendants(first, 10, remembered), {10, 11, 12})
        later = {10: first[10], 11: {'ppid': 1, 'start': 999}, 12: {'ppid': 1, 'start': 12}}
        self.assertEqual(runner.descendants(later, 10, remembered), {10, 12})

    def test_only_declared_cpu_classes_are_excluded(self):
        base = {'ppid': 1, 'comm': 'unrelated', 'cgroup': '0::/service'}
        self.assertEqual(runner.cpu_class(8, base, set(), {'abc'}), 'foreign')
        self.assertEqual(runner.cpu_class(8, base, {8}, {'abc'}), 'runner_descendants')
        self.assertEqual(runner.cpu_class(8, dict(base, cgroup='0::/docker/abc'), set(), {'abc'}), 'own_containers')
        self.assertEqual(runner.cpu_class(8, dict(base, ppid=2), set(), set()), 'kernel_threads')
        for name in ['dockerd', 'containerd', 'containerd-shim', 'containerd-shim-runc-v2', 'docker-proxy', 'runc']:
            self.assertEqual(runner.cpu_class(8, dict(base, comm=name), set(), set()), 'container_runtime')
        self.assertEqual(runner.cpu_class(8, dict(base, comm='postgres'), set(), set()), 'foreign')


if __name__ == '__main__':
    unittest.main()
