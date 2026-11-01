"""D-120 measurement-end regression proofs; no sockets or privileged tools."""
from copy import deepcopy
from contextlib import ExitStack
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from reload_analysis import classify_d120
from stream import TraceEnd, main
from trace_capture import final_guard, finish_stream


def window():
    return dict(generation='1100', verified=True, upper_ns=30_000_000,
                render_to_installed_ns=20_000_000,
                install_start={'wall_ns': '1000000000'},
                install_end={'wall_ns': '1010000000'},
                release_start={'wall_ns': '1020000000'},
                release_end={'wall_ns': '1030000000'})


def synthetic_stream(successor='ok'):
    """Execute the real stream loop with a stop request during its lost read."""
    class Finished(BaseException):
        pass

    with tempfile.TemporaryDirectory() as tmp:
        path = Path(tmp) / 'stream'
        tokens = []

        class Socket:
            def settimeout(self, value):
                assert value == 1.5

            def connect(self, address):
                pass

            def sendall(self, token):
                self.token = token

            def recv(self, size):
                return self.token

            def sendto(self, token, address):
                tokens.append(token)

            def getsockname(self):
                return ('192.0.2.1', 40000)

            def recvfrom(self, size):
                if len(tokens) == 1:
                    # Request arrives while the final UDP read is outstanding.
                    Path(str(path) + '.trace-stop').touch()
                    raise TimeoutError()
                if successor == 'TimeoutError':
                    raise TimeoutError()
                return (tokens[-1] if successor == 'ok' else b'wrong', ('203.0.113.10', 9000))

        def sleep(_):
            if Path(str(path) + '.trace-done').exists():
                raise Finished()

        with ExitStack() as opened, \
                patch('stream.open', side_effect=lambda *a, **kw: opened.enter_context(open(*a, **kw)), create=True), \
                patch('stream.sys.argv', ['stream.py', '203.0.113.10', '9001', '9000', str(path)]), \
                patch('stream.socket.socket', side_effect=lambda *args: Socket()), \
                patch('stream.time.sleep', side_effect=sleep):
            try:
                main()
            except Finished:
                pass
        events = [json.loads(line) for line in Path(str(path) + '.jsonl').read_text().splitlines()]
        reads = {e['sequence']: e for e in events if e['event'] == 'udp_read'}
        ds = [dict(sequence=e['sequence'], send=e, read=reads[e['sequence']],
                   stages={'node_ingress': {'wall_ns': 1_015_000_000}})
              for e in events if e['event'] == 'udp_send']
        return dict(events=events, old_end=classify_d120(ds[:1], [window()]),
                    new_end=classify_d120(ds, [window()]),
                    done=json.loads(Path(str(path) + '.trace-done').read_text()))


def campaign():
    def stamp(n):
        return dict(wall_ns=str(n), mono_ns=str(n))
    return [dict(event='apply', generation=str(g), start=stamp(1), end=stamp(5),
                 verified=True, deadline=dict(render_start_ns='1', installed_by_ns='2'),
                 batches=[dict(kind='guard_install', start=stamp(1), end=stamp(2), ok=True),
                          dict(kind='guard_release', start=stamp(3), end=stamp(4), ok=True)])
            for g in range(1000, 1101)]


class D120EndTest(unittest.TestCase):
    def test_pending_loss_gets_exactly_its_successor(self):
        result = synthetic_stream()
        self.assertFalse(result['old_end']['passed'])
        self.assertEqual(result['old_end']['losses'][0]['next_datagram'], 'missing')
        self.assertTrue(result['new_end']['passed'], result)
        self.assertEqual(result['done'], dict(end_reason='complete', extra_cycles=1, sequence=2))
        self.assertEqual([e['sequence'] for e in result['events'] if e['event'] == 'udp_send'], [1, 2])

    def test_failed_successor_is_never_retried_and_bound_stays_missing(self):
        for outcome in ('TimeoutError', 'wrong_echo'):
            with self.subTest(outcome=outcome):
                result = synthetic_stream(outcome)
                self.assertFalse(result['new_end']['passed'])
                self.assertEqual([r['next_datagram'] for r in result['new_end']['losses']], ['failed', 'missing'])
                self.assertEqual(result['done'], dict(end_reason='missing', extra_cycles=1, sequence=2))
                self.assertEqual([e['sequence'] for e in result['events'] if e['event'] == 'udp_send'], [1, 2])

    def test_tcp_failure_in_extra_cycle_cannot_hide_missing_udp_successor(self):
        end = TraceEnd()
        end.udp_read('TimeoutError')
        self.assertIsNone(end.boundary(True))
        # No UDP read occurred because the TCP operation failed.
        self.assertEqual(end.boundary(True), 'missing')

    def test_earlier_losses_with_successors_need_no_extra_cycle(self):
        end = TraceEnd()
        end.udp_read('TimeoutError')
        self.assertIsNone(end.boundary(False))
        end.udp_read('ok', True)
        self.assertEqual(end.boundary(True), 'complete')
        self.assertEqual(end.extra_cycles, 0)

    def test_no_request_never_ends_or_spends_extension(self):
        end = TraceEnd()
        for outcome in ('ok', 'TimeoutError'):
            end.udp_read(outcome, outcome == 'ok')
            for _ in range(3):
                self.assertIsNone(end.boundary(False))
        self.assertEqual(end.extra_cycles, 0)

    def test_final_guard_required_before_stop_request(self):
        good = campaign()
        missing_release = deepcopy(good)
        missing_release[-1]['batches'].pop()
        failed_release = deepcopy(good)
        failed_release[-1]['batches'][-1]['ok'] = False
        unverified = deepcopy(good)
        unverified[-1]['verified'] = False
        not_closed = deepcopy(good)
        not_closed[-1]['end']['mono_ns'] = '3'
        with tempfile.TemporaryDirectory() as tmp:
            apply = Path(tmp) / 'apply'
            stream = str(Path(tmp) / 'stream')
            for rows in (good[:-1], missing_release, failed_release, unverified, not_closed):
                apply.write_text(''.join(json.dumps(row) + '\n' for row in rows))
                with self.assertRaises(ValueError):
                    finish_stream(stream, apply)
                self.assertFalse(Path(stream + '.trace-stop').exists())
            apply.write_text(''.join(json.dumps(row) + '\n' for row in good))
            self.assertEqual(final_guard(apply)['generation'], '1100')
            Path(stream + '.trace-done').write_text('{}\n')
            finish_stream(stream, apply)
            receipt = json.loads(Path(stream + '.trace-finish.json').read_text())
            self.assertEqual(receipt['max_extra_cycles'], 1)
            self.assertTrue(Path(stream + '.trace-stop').exists())

    def test_watchdog_does_not_fabricate_successor_or_boundary(self):
        with tempfile.TemporaryDirectory() as tmp:
            apply = Path(tmp) / 'apply'
            stream = str(Path(tmp) / 'stream')
            apply.write_text(''.join(json.dumps(row) + '\n' for row in campaign()))
            with patch('trace_capture.time.monotonic', side_effect=[0, 16]):
                with self.assertRaisesRegex(RuntimeError, 'successor missing'):
                    finish_stream(stream, apply)
            self.assertFalse(Path(stream + '.trace-done').exists())
            self.assertFalse(Path(stream + '.jsonl').exists())


if __name__ == '__main__':
    unittest.main()
