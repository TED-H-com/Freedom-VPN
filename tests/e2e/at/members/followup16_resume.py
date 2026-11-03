"""Post-reboot joins of retained evidence only; never starts a stack or probe."""
from collections import Counter
from datetime import datetime, timezone
from decimal import Decimal
import hashlib
import json
from pathlib import Path
import re
import sys

from evidence_io import load, publish, read

ROOT = Path('docs/e2e/at/members')
RUN = '0ffb355a-945b-4198-a551-3794120a686c'
CAP = ROOT/'history/follow-up-16-captures'/RUN
START = '2026-11-01T17:32:43Z'
END = '2026-11-01T17:37:12Z'
CONTEXT = dict(
    classification='contention-confounded for every observation overlapping the window',
    start=START, end=END, decision='D-133', cause_status='unproven',
    source='Orchestrator resume instruction, 2026-11-02: handover 29 CPU-heavy gates',
    rule='All timings, failures and anomalies inside or overlapping this window are contention-confounded. Values and verdicts stand; overlap assigns no cause. Cumulative snapshots may reference earlier events; evaluate the event time, not the later snapshot time.',
    index='F16-contention.json',
)


def save(path, data):
    path.write_text(json.dumps(data, indent=2)+'\n')
    publish(path)


def utc_ns(value):
    # Preserve ns without floating point, including records with -05:00 offsets.
    match = re.fullmatch(r'(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d)(?:\.(\d+))?(Z|[+-]\d\d:\d\d)', value)
    if not match:
        return None
    base = datetime.fromisoformat(match[1]+match[3].replace('Z', '+00:00'))
    delta = base-datetime(1970, 1, 1, tzinfo=timezone.utc)
    return (delta.days*86400+delta.seconds)*10**9+int(((match[2] or '')+'000000000')[:9])


def review():
    # Join every observed ::1-source structural WG message in the retained captures.
    rows = []
    for directory in sorted((CAP/'headers').iterdir()):
        packets = {}
        for role in ['member', 'node']:
            p = directory/(role+'.headers.jsonl')
            if p.exists():
                packets[role] = [json.loads(line) for line in p.read_text().splitlines()]
        cap = load(CAP/(directory.name+'-original-capture.json'))
        messages = sorted({p['wireguard']['message_sha256'] for pp in packets.values() for p in pp if p.get('ip_version')==6 and p.get('wireguard', {}).get('source_loopback')})
        for digest in messages:
            hops = [dict(role=role, packet=p) for role, pp in packets.items() for p in pp if p.get('wireguard', {}).get('message_sha256')==digest]
            samples = []
            for sample in cap['samples']:
                samples.append(dict(label=sample['label'], points={role:dict(
                    start=p['start'], end=p['end'],
                    latest_handshake_unix_seconds=p.get('latest_handshake_unix_seconds'),
                    wireguard_endpoint=p.get('wireguard_endpoint'), routes=p.get('routes'),
                    ipv6_counters=p.get('kernel', {}).get('snmp6', {}))
                    for role, p in sample['points'].items() if role in packets}))
            rows.append(dict(capture=directory.name, capture_id=cap['id'], message_sha256=digest,
                             matching_hops=hops, independent_samples=samples, cause_status='unproven'))
    first = load(CAP/'initial-selection-analysis.json')
    syn = first['member_syns'][0]
    arrival = next(p for p in first['hops']['node'] if p['interface']=='fvmb0' and p['packet_type']!=4 and p.get('tcp')==syn['tcp'])
    delayed = dict(member_first_syn=syn, node_matching_first_syn=arrival,
                   transit_lower_ns=str(int(arrival['mono_lower_ns'])-int(syn['mono_upper_ns'])),
                   transit_upper_ns=str(int(arrival['mono_upper_ns'])-int(syn['mono_lower_ns'])),
                   original_request_result='No authenticated origin receipt before the five-second deadline; original failed attempt unchanged',
                   scope='The longer native capture sees late delivery after the original AF_PACKET/HTTP request window. Late SYNs and SYN-ACKs are not a successful HTTP receipt.')
    save(ROOT/'F16-loopback-and-first-selection.json', dict(run_id=RUN, cause_status='unproven',
         ipv6_loopback_messages=rows, first_selection=delayed,
         scope='Structural WireGuard header/index/hash joins are not cryptographic authentication. Sampled IPv6 counter changes do not prove the exact kernel discard site. No cause assigned to F11-01, F14-01 or AT02. All four AT02/AT03 full-session captures contain no IPv6 ::1-source WG packet; IPv4 loopback type-1 traffic remains indexed separately.'))

    case = load(ROOT/'AT09.json')
    failures = [s for s in case['steps'] if s['status']=='fail']
    assert len(failures)==4
    explanations = [
        'Original member-session GET returned 401, exits=0. Session expiry is a hypothesis: no exact expiry reason was retained. Re-authentication before this long-campaign check is a future-run harness fix; original failure stands.',
        'The predicate includes non-serving safety reports (tunnel_probe HEALTH_CHECK_FAILED and TC_FOREIGN_FILTER/read_failed), not only awaiting_member_grants provisional reports. It therefore failed revision-11 matching. No predicate relaxation or D-067 pass is inferred.',
        'Same-generation 64 -> 64 restart retains a populated seven-member lease and first counted recovery at 11.776274584 s; the combined assertion also requires the failed provisional-observation predicate. Neither the latency nor historical failed rows are waived. Earlier injected TC fault was removed; this run does not prove it caused the later read_failed report.',
        'Receive-only dual MTU 1280 remains FAIL: 12,012 bytes in 12,011 ms. Asymmetric harness setup shows size-selective loss without an observed generated PTB, not proof that a generated PTB was lost. Evidence for D-137 only.',
    ]
    out = []
    for step, explanation in zip(failures, explanations):
        out.append(dict(original_step=step, review=explanation, cause_status='unproven',
                        contention='outside the reported 17:32:43-17:37:12 UTC window'))
    save(ROOT/'F16-AT09-failure-review.json', dict(run_id=RUN, source_commit=case['source_commit'],
         original_status=case['status'], failure_count=len(out), failures=out,
         historical_statuses='unchanged', no_new_run=True))


def contention():
    # Records remain intact. Add context to living records and retain an exhaustive
    # pointer index into their timestamped fields, including JSON stored in actual.
    lo, hi = utc_ns(START), utc_ns(END)
    offsets = []
    for p in CAP.glob('*-analysis.json*'):
        if p.name.endswith('.sha256.json'):
            continue
        d = load(p)
        for clock in d.get('clock_offsets', {}).values():
            if clock.get('boot_id')=='97c018d7-a925-4723-9127-83c097de1c8a':
                offsets.extend([int(clock['offset_lower_ns']), int(clock['offset_upper_ns'])])
    offset_lo, offset_hi = min(offsets), max(offsets)
    # Empirical envelope under the retained no-wall-step assumption, conservative
    # at the boundary; elapsed durations are not mistaken for absolute clocks.
    mono_lo, mono_hi = lo+offset_lo, hi+offset_hi
    indexed = []
    def times(obj, pointer=''):
        if isinstance(obj, dict):
            for k, v in obj.items():
                if k=='contention_context':
                    continue
                yield from times(v, pointer+'/'+k.replace('~', '~0').replace('/', '~1'))
        elif isinstance(obj, list):
            for i, v in enumerate(obj):
                yield from times(v, pointer+'/'+str(i))
        elif isinstance(obj, (str, int)) and not isinstance(obj, bool):
            key = pointer.rsplit('/', 1)[-1]
            if isinstance(obj, str) and obj[:1] in ['{', '[']:
                try:
                    parsed = json.loads(obj)
                except ValueError:
                    return
                yield from times(parsed, pointer+'/@json')
                return
            if isinstance(obj, str) and obj.startswith('2026-'):
                value = utc_ns(obj)
                if value is not None and lo <= value <= hi:
                    yield dict(pointer=pointer, clock='UTC', value=obj)
            elif key=='timestamp':
                try:
                    value = int(Decimal(str(obj))*10**9)
                except ArithmeticError:
                    return
                if lo <= value <= hi:
                    yield dict(pointer=pointer, clock='pcap epoch seconds', value=str(obj))
            elif key.endswith('_ns') or key=='ns':
                try:
                    value = int(obj)
                except ValueError:
                    return
                if mono_lo <= value <= mono_hi:
                    yield dict(pointer=pointer, clock='same-boot monotonic ns', value=str(obj))
                elif lo <= value <= hi:
                    yield dict(pointer=pointer, clock='epoch ns', value=str(obj))
    candidates = sorted(p for p in ROOT.iterdir() if p.is_file() and
                        (re.fullmatch(r'AT\d\d(?:-enforcement-timing)?\.json(?:\.gz)?', p.name) or p.name.startswith('F16-') and p.name.endswith(('.json', '.json.gz'))) and not p.name.endswith('.sha256.json') and not p.name.startswith('F16-contention'))
    case_rows = []
    for path in candidates:
        d = load(path)
        if not isinstance(d, dict):
            continue
        d['contention_context'] = CONTEXT
        if re.fullmatch(r'AT\d\d\.json(?:\.gz)?', path.name):
            for i, step in enumerate(d['steps']):
                in_window = lo <= utc_ns(step['at']) <= hi
                refs = list(times(step))
                if in_window or refs:
                    step['contention_context'] = dict(
                        classification='contention-confounded' if in_window else 'contains earlier contention-confounded observations',
                        step_completed_in_window=in_window, referenced_times_in_window=len(refs),
                        index='F16-contention.json', cause_status='unproven')
                    case_rows.append(dict(case=d['id'], step_index=i, name=step['name'],
                                          status=step['status'], at=step['at'], **step['contention_context']))
        destination = Path(str(path).removesuffix('.gz'))
        save(destination, d)
    paths = candidates+sorted(p for p in CAP.rglob('*') if p.is_file() and p.name.endswith(('.json', '.jsonl', '.json.gz', '.jsonl.gz')) and not p.name.endswith('.sha256.json'))
    inventory = []
    indexpath = ROOT/'F16-contention-observations.jsonl'
    with indexpath.open('w') as output:
        for path in paths:
            raw = read(path)
            name = str(path.relative_to(ROOT))
            if path.name.endswith(('.jsonl', '.jsonl.gz')):
                documents = ((json.loads(line), '/line/'+str(i+1)) for i, line in enumerate(raw.splitlines()))
            else:
                documents = [(json.loads(raw), '')]
            count = 0
            for doc, pointer in documents:
                for match in times(doc, pointer):
                    output.write(json.dumps(dict(file=name, classification='contention-confounded', cause_status='unproven', **match), separators=(',', ':'))+'\n')
                    count += 1
            inventory.append(dict(file=name, uncompressed_sha256=hashlib.sha256(raw).hexdigest(), marked_timestamp_references=count))
    publish(indexpath)
    captures = []
    for path in sorted(CAP.glob('*-original-capture.json')):
        cap = load(path)
        a = min(utc_ns(p['launch_before']['wall']) for p in cap['points'].values())
        b = max(utc_ns(p['stopped_after']['wall']) for p in cap['points'].values())
        overlap = a <= hi and b >= lo
        captures.append(dict(file=str(path.relative_to(ROOT)), capture_id=cap['id'],
                             start_wall_ns=str(a), end_wall_ns=str(b), overlaps=overlap,
                             classification='contention-confounded for the overlapping portion' if overlap else 'outside this reported window'))
    save(ROOT/'F16-contention.json', dict(run_id=RUN, **CONTEXT,
         monotonic_window=dict(boot_id='97c018d7-a925-4723-9127-83c097de1c8a', start_ns=str(mono_lo), end_ns=str(mono_hi),
                               assumption='Envelope of retained native wall/monotonic clock pairs; no intervening wall-clock step. Boundary overlap conservatively included.'),
         steps=case_rows, capture_windows=captures, files=inventory,
         timestamp_references=sum(r['marked_timestamp_references'] for r in inventory),
         observations='F16-contention-observations.jsonl'+('.gz' if not indexpath.exists() else ''),
         coverage='All root F16/AT JSON and this run\'s retained capture JSON/JSONL; @json traverses serialized actual fields, /line is one-based JSONL. This conservatively marks historical references too, without asserting that a later snapshot was measured during contention. Intervals crossing the window are confounded even if endpoints lie outside. No known time is inferred for untimed records. Earlier archives unchanged.'))
    print('contention:', len(inventory), 'files,', sum(r['marked_timestamp_references'] for r in inventory), 'timestamp references,', len(case_rows), 'steps')


if __name__=='__main__':
    if sys.argv[1:]==['contention']:
        contention()
    else:
        review()
