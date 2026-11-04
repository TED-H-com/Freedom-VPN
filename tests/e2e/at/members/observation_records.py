"""Read agent-core 52's documented receipts; diagnostic only, never a verdict.

Input logs stay private. Publish only the documented metadata, stable boot labels,
line references and byte digests. Attempts are per report per process, not durable
outbox counters. A completion receipt is not an exact server-acceptance timestamp.
"""
import calendar
import datetime
import hashlib
import json
import re
import sys
from decimal import Decimal

FIELDS = re.compile(r'(?:^|\s)([a-z_]+)=("(?:\\.|[^"\\])*"|\[[^\]]*\]|[^\s]+)')
REASONS = {'closed_recovery_binding', 'first_report', 'serving_changed', 'state_changed', 'interval_elapsed'}
OUTCOMES = {'accepted', 'rejected', 'transport_error', 'timeout', 'unconfirmed_response'}
CLASSES = {'', 'deadline_exceeded', 'network_timeout', 'canceled', 'connection_refused', 'connection_reset', 'connection_closed', 'dns', 'network', 'other'}
UUID = re.compile(r'^[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}$')
STAMP = re.compile(r'^(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d)(?:\.(\d{1,9}))?(Z|\+00:00)$')


def ns(value):
    m = STAMP.fullmatch(value)
    if not m:
        raise ValueError('timestamp format')
    t = datetime.datetime.strptime(m[1], '%Y-%m-%dT%H:%M:%S')
    return calendar.timegm(t.timetuple()) * 10**9 + int((m[2] or '').ljust(9, '0'))


def seconds(n):
    return str(Decimal(n) / Decimal(10**9))


def ids(value):
    if isinstance(value, str):
        if not (value.startswith('[') and value.endswith(']')):
            raise ValueError('ID array')
        value = value[1:-1].split()
    if not isinstance(value, list) or any(not isinstance(x, str) or not UUID.fullmatch(x) for x in value):
        raise ValueError('report ID')
    return value


def records(raw, source):
    result, times, malformed = [], [], []
    for line_number, line in enumerate(raw.splitlines(), 1):
        try:
            if line.startswith('{'):
                f = json.loads(line)
            else:
                f = {k: json.loads(v) if v.startswith('"') else v for k, v in FIELDS.findall(line)}
            at = f.get('time', '')
            ns(at)
            times.append(at)
            if f.get('msg') not in ('observation enqueued', 'observation submission'):
                continue
            r = dict(time=at, msg=f['msg'], source=source, line=line_number,
                     line_sha256=hashlib.sha256(line.encode()).hexdigest())
            if f['msg'] == 'observation enqueued':
                if not UUID.fullmatch(f['report_id']) or f['due_reason'] not in REASONS:
                    raise ValueError('enqueue metadata')
                ns(f['observed_at'])
                r.update({k: f[k] for k in ('report_id', 'observed_at', 'due_reason')})
            else:
                report_ids, accepted = ids(f['report_ids']), ids(f['accepted_report_ids'])
                attempts = f['attempt_numbers']
                if isinstance(attempts, str):
                    attempts = attempts.strip('[]').split()
                attempts = [int(x) for x in attempts]
                if len(attempts) != len(report_ids) or any(x < 1 for x in attempts):
                    raise ValueError('attempt array')
                if f['attempt_scope'] != 'process' or f['outcome'] not in OUTCOMES or f['transport_error_class'] not in CLASSES:
                    raise ValueError('submission enum')
                if f['error_code'] and not re.fullmatch(r'[A-Z][A-Z0-9_]*|unknown', f['error_code']):
                    raise ValueError('error code')
                ns(f['started_at'])
                retry = f['next_retry_at']
                if retry in (None, '<nil>'):
                    retry = None
                else:
                    ns(retry)
                r.update(report_ids=report_ids, attempt_numbers=attempts,
                         attempt_scope='process', process_id=int(f['process_id']),
                         boot_id='boot-' + hashlib.sha256(str(f['boot_id']).encode()).hexdigest()[:16],
                         started_at=f['started_at'], duration_ms=int(f['duration_ms']),
                         outcome=f['outcome'], accepted_report_ids=accepted,
                         http_status=int(f['http_status']), error_code=f['error_code'],
                         transport_error_class=f['transport_error_class'], next_retry_at=retry,
                         retry_after_ms=int(f['retry_after_ms']))
                if min(r['process_id'], r['duration_ms'], r['retry_after_ms']) < 0:
                    raise ValueError('negative metadata')
            result.append(r)
        except (ValueError, KeyError, TypeError):
            if 'observation enqueued' in line or 'observation submission' in line:
                malformed.append(dict(source=source, line=line_number, line_sha256=hashlib.sha256(line.encode()).hexdigest()))
    return result, times, malformed


def extract(before, after, start, end, observations):
    a, b = ns(start), ns(end)
    before_records, before_times, before_bad = records(before, 'start_export')
    after_records, after_times, after_bad = records(after, 'end_export')
    prefix = after.startswith(before)
    # Keep the earlier export if a restart overwrote the file. Never invent the
    # lost interval between that snapshot and the surviving suffix.
    all_records = after_records if prefix else before_records + after_records
    unique = {}
    for r in all_records:
        unique[r['line_sha256']] = r
    all_records = sorted(unique.values(), key=lambda r: ns(r['time']))
    enqueues = [r for r in all_records if r['msg'] == 'observation enqueued']
    attempts = [r for r in all_records if r['msg'] == 'observation submission']
    window_enqueues = [r for r in enqueues if a <= ns(r['time']) < b]
    window_attempts = [r for r in attempts if ns(r['started_at']) < b and ns(r['time']) >= a]
    stored = [o for o in observations if a <= ns(o['received_at']) < b]
    joins, missing = [], []
    for o in stored:
        rid = o['report_id']
        es = [r for r in enqueues if r['report_id'] == rid]
        ats = [r for r in attempts if rid in r['report_ids']]
        accepted = [r for r in ats if r['outcome'] == 'accepted' and rid in r['accepted_report_ids']]
        e = es[0] if es else None
        ac = accepted[0] if accepted else None
        absent = []
        if not e:
            absent.append('enqueue')
        if not ac:
            absent.append('accepted_attempt')
        if absent:
            missing.append(dict(report_id=rid, missing=absent))
        joins.append(dict(observation=o, enqueue=e, attempts=ats, accepted_attempts=accepted,
                          enqueue_to_attempt_seconds=seconds(ns(ac['started_at'])-ns(e['time'])) if e and ac else None,
                          enqueue_to_accepted_completion_seconds=seconds(ns(ac['time'])-ns(e['time'])) if e and ac else None,
                          enqueue_to_stored_seconds=seconds(ns(o['received_at'])-ns(e['time'])) if e else None,
                          observed_to_stored_seconds=seconds(ns(o['received_at'])-ns(e['observed_at'])) if e else None,
                          attempt_to_stored_seconds=seconds(ns(o['received_at'])-ns(ac['started_at'])) if ac else None))
    first = min(after_times, key=ns) if after_times else None
    last = max(after_times, key=ns) if after_times else None
    suffix = max(a, ns(first)) if first else None
    return dict(format_document='docs/node/README.md: observation enqueued / observation submission',
                submission_events_available=bool(attempts), enqueues=window_enqueues,
                submission_attempts=window_attempts, stored_observation_joins=joins,
                enqueued_not_stored_in_window=[r for r in window_enqueues if r['report_id'] not in {o['report_id'] for o in stored}],
                stored_observations_missing_records=missing,
                malformed_receipts=before_bad+after_bad,
                window_start=start, window_end=end,
                start_bytes=str(len(before.encode())), end_bytes=str(len(after.encode())),
                start_sha256=hashlib.sha256(before.encode()).hexdigest(), end_sha256=hashlib.sha256(after.encode()).hexdigest(),
                start_content_retained_as_prefix=prefix, overwrite_or_replacement_detected=not prefix,
                first_retained_log_time=first, last_retained_log_time=last,
                timestamped_lines_in_window=sum(a <= ns(t) < b for t in after_times),
                row_duration_seconds=seconds(b-a),
                surviving_end_export_window_seconds=seconds(max(0, b-suffix)) if suffix is not None else None,
                missing_prefix_seconds=seconds(min(b, suffix)-a) if suffix is not None and not prefix else ('0' if prefix else None),
                start_export_last_log_time=max(before_times, key=ns) if before_times else None,
                coverage_limit='Log times bound retained lines, not silent periods. Overwrite loses unsnapshotted lines. A killed request may lack completion; a row ending before completion may lack its receipt. Not stored means absent from this window, not globally rejected. Log time has millisecond precision; observed_at/started_at retain nanoseconds. boot_id is a stable SHA-256 label; process_id and attempt_scope are retained. accepted completion is the client receipt, received_at is Central storage, neither proves applied verification.')


if __name__ == '__main__':
    v = json.load(sys.stdin)
    json.dump(extract(v['before'], v['after'], v['start'], v['end'], v['observations']), sys.stdout)
    print()
