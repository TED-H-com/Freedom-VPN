"""ISO-F6-01 pre-install failures; never installed-verifier evidence."""
import calendar
from datetime import datetime
import re
from producer_timing import command_lifecycle, identity


def wall_ns(value):
    base, _, fraction = value.removesuffix('Z').partition('.')
    if not value.endswith('Z') or len(fraction) > 9:
        raise ValueError('UTC nanosecond timestamp required')
    return calendar.timegm(datetime.strptime(base, '%Y-%m-%dT%H:%M:%S').timetuple()) * 10**9 + int(fraction.ljust(9, '0'))


def candidate_rejections(transactions, operations, initial_generation, start, end):
    """Join accepted dirty reports to the last completed failed candidate attempt.

    The error timestamp follows Apply's detached snapshot; the next Apply start
    bounds that join. Both clocks are retained, with no float/time allowance.
    """
    lo, hi = wall_ns(start), wall_ns(end)
    applies = [r for r in operations if r.get('timing', {}).get('operation') == 'apply']
    result = []
    for index, wire in enumerate(transactions):
        body = wire.get('body') or {}
        if (not wire.get('path', '').endswith('/observations') or wire.get('status') != 202
                or body.get('member_forwarding_enabled') is not False):
            continue
        if not all(lo <= wall_ns(wire[k]) < hi for k in ('received_at', 'response_at')):
            continue
        if not lo <= wall_ns(body['observed_at']) < hi:
            continue
        for error_index, error in enumerate(body.get('errors') or []):
            match = re.fullmatch(r'candidate_generation=([1-9][0-9]*) verified=false', error.get('message', ''))
            if error.get('stage') != 'apply_candidate' or error.get('code') != 'TC_FOREIGN_FILTER' or not match:
                continue
            generation = match[1]
            at = wall_ns(error['observed_at'])
            if int(generation) <= int(initial_generation) or not lo <= at <= wall_ns(body['observed_at']) < hi:
                continue
            # The candidate must have been delivered as an original desired body
            # during this injection, rather than inferred from an old applied ACK.
            desired_indices = [i for i,w in enumerate(transactions) if 'desired-state' in w.get('path','') and w.get('status') == 200 and (w.get('response') or {}).get('signed',{}).get('config_generation') == generation and lo <= wall_ns(w['response_at']) <= at]
            if not desired_indices:
                continue
            preceding = [r for r in applies if wall_ns(r['anchor']['wall']) <= at]
            if not preceding:
                continue
            attempt = max(preceding, key=lambda r: wall_ns(r['anchor']['wall']))
            t = attempt['timing']
            if t['candidate_generation'] != generation or t['succeeded'] or t['verified'] or t.get('verifications'):
                continue
            # A failed preparation has no installed/rules-verifier evidence.
            if t['deadline']['install_ok'] or t['deadline']['guard_released']:
                continue
            later = [r for r in applies if r['run_id'] == attempt['run_id'] and int(r['sequence']) > int(attempt['sequence'])]
            if later:
                nxt = min(later, key=lambda r: int(r['sequence']))
                earliest_start = wall_ns(nxt['anchor']['wall']) + int(nxt['timing']['start_ns']) - int(nxt['anchor']['mono_after_ns'])
                if at >= earliest_start:
                    continue
            commands = command_lifecycle(attempt)
            if commands['status'] != 'available' or not commands['records']:
                continue
            result.append(dict(transaction_index=index, report_id=body['report_id'], report_observed_at=body['observed_at'],
                accepted_at=wire['response_at'], applied_ack=body.get('applied_config_generation'), candidate_generation=generation,
                error_index=error_index, error=error, desired_transaction_indices=desired_indices,
                attempt=identity(attempt), original_command_indices=list(range(len(t['commands']))),
                command_lifecycle=commands, evidence_kind='pre-install rejection; installed verifier unavailable',
                attempt_anchor=attempt['anchor']))
    return result
