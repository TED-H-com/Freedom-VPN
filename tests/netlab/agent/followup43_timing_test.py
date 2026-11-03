"""TEST ONLY synthetic process tables; no Docker or host mutation."""
import unittest
import followup43_overlay as overlay
import followup43_timing as runner


class QualificationTests(unittest.TestCase):
    def test_container_events_retain_even_transient_foreign_activity(self):
        state = runner.ContainerState([{'id': 'shared', 'name': 'fvpn-pg'}])
        def event(identity, name, action):
            return {'Action': action, 'Actor': {'ID': identity, 'Attributes': {'name': name}}}
        state.update(event('own', runner.PREFIX, 'start'))
        state.update(event('other', 'fvpn-another-lane', 'start'))
        state.update(event('other', 'fvpn-another-lane', 'die'))
        running, foreign, own = state.snapshot()
        self.assertEqual({r['id'] for r in running}, {'shared', 'own'})
        self.assertEqual(foreign, [{'id': 'other', 'name': 'fvpn-another-lane'}])
        self.assertEqual(own, {'own'})
        self.assertEqual(state.snapshot()[1], [])
        state.update(event('own', runner.PREFIX, 'die'))
        self.assertEqual(state.snapshot()[2], {'own'})

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


class OverlayTests(unittest.TestCase):
    def test_hooks_only_rename_step_and_bracket_pre_apply(self):
        original = ('func (e *Engine) Step(ctx context.Context) error {\n'
                    '\twasServing := e.expectServing.Load()\n'
                    '\tres, err := e.plane.Apply(applyCtx, model, prev)\n}')
        got = overlay.instrument(original)
        self.assertIn('f43OriginalStep(ctx context.Context)', got)
        self.assertLess(got.index('e.f43Span("pre_apply_accounting")'), got.index('wasServing :='))
        self.assertLess(got.index('f43Accounting.stop()'), got.index('e.plane.Apply('))
        self.assertIn('defer f43Accounting.write(e)', got)
        with self.assertRaises(ValueError):
            overlay.instrument(original + original)
        with self.assertRaises(ValueError):
            overlay.instrument(original.replace('Step(', 'Other('))


if __name__ == '__main__':
    unittest.main()
