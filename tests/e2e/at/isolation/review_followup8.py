#!/usr/bin/env python3
"""Derive the six G2 assertions from retained real-agent wire and kernel reads."""
import hashlib
import json
from evidence_io import read_json
from pathlib import Path
import re
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else 'docs/e2e/at/isolation')
def read(name, default=None):
    p = root / name
    return read_json(p) if p.exists() or Path(str(p)+'.gz').exists() else default

def write(name, value):
    (root / name).write_text(json.dumps(value, indent=2) + '\n')

receiver = read('receiver-capture.json', {})
transactions = receiver.get('transactions', [])
reports = []
for r in transactions:
    if r.get('path', '').endswith('/observations') and isinstance(r.get('body'), dict):
        reports.append(dict(report=r['body'], received_at=r['received_at'], status=r['status']))
trace = read('foreign-redirect.json', {})
rows = trace.get('rows', [])
def labelled(label):
    return [r for r in rows if r['label'] == label]
def first(label):
    rs = labelled(label)
    return rs[0] if rs else {}
def member_up(row):
    try:
        links=json.loads(row.get('readback','').splitlines()[0])
        return next(('UP' in link.get('flags',[]) for link in links if link.get('ifname')=='fvmb0'), None)
    except (ValueError, IndexError, TypeError):
        return None

def current_tc(row):
    # The first five command results are links/addresses/routes/qdiscs/filters.
    # The later heartbeat intentionally contains HISTORICAL rejected rows.
    try:
        lines=row.get('readback','').splitlines()
        qdiscs,filters=json.loads(lines[3]),json.loads(lines[4])
        if not isinstance(qdiscs,list) or not isinstance(filters,list):
            return ''
        return json.dumps(dict(qdiscs=qdiscs,filters=filters))
    except (ValueError, IndexError, TypeError):
        return ''

def tc(report):
    return report.get('tc_evidence') or {}
def qos(report):
    return next((c.get('detail', '') for c in report.get('checks', []) if c['name'] == 'qos'), '')
def digest(t):
    if not t:
        return False
    projection = dict(reads=t['reads_total'], objects=t['objects_total'], truncated=t['truncated'], tc=t['attachments'])
    # Contract projection contains ASCII tokens, arrays, booleans and small integers:
    # this sorted compact JSON is identical to RFC 8785 for that restricted domain.
    raw = json.dumps(projection, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()
    return hashlib.sha256(raw).hexdigest() == t['sha256']

initial = first('before injection').get('observation', {})
initial_gen = int(initial.get('applied_config_generation') or 0)
injection = first('inject clsact mirred')
removal = first('remove all injected objects')
clean_request = first('publish clean newer generation')
start, end = injection.get('start', '9999'), removal.get('end', '9999')
rejections = [r for r in reports if start <= r['report'].get('observed_at', '') < end and tc(r['report']).get('error_code') == 'TC_FOREIGN_FILTER']
clean = [r for r in reports if r['report'].get('observed_at', '') >= clean_request.get('at', '9999') and tc(r['report']).get('verified') is True and tc(r['report']).get('phase') == 'verify' and int(tc(r['report']).get('candidate_generation') or 0) > initial_gen and tc(r['report']).get('read_at', '') >= end and tc(r['report']).get('error_code') is None and tc(r['report']).get('rejected') is None]
apply = [r for r in clean if 'source=apply' in qos(r['report']) and 'verified=true' in qos(r['report'])]
live = [r for r in clean if 'source=live' in qos(r['report']) and 'verified=true' in qos(r['report'])]
matching = [r for r in clean if r['report'].get('member_forwarding_enabled') is True and (r['report'].get('lease') or {}).get('config_generation') == r['report'].get('applied_config_generation') == tc(r['report']).get('candidate_generation')]
# Some wire lease summaries omit generation; resolve them through signed responses.
leases = {r['response']['signed']['lease_id']:r['response']['signed'] for r in transactions if 'lease' in r.get('path', '') and isinstance(r.get('response'), dict) and 'signed' in r['response']}
for r in clean:
    b=r['report']; l=leases.get((b.get('lease') or {}).get('lease_id'), {})
    if b.get('member_forwarding_enabled') is True and l.get('members') and l.get('config_generation') == b.get('applied_config_generation') == tc(b).get('candidate_generation') and r not in matching:
        matching.append(r)
negative = labelled('dirty newer generation')
dirty_reports = [r.get('observation', {}) for r in negative]
# Agent-core 32 identifies a rejected candidate separately from the installed
# verifier. Only assertion 6 consumes this pre-install evidence; 1 and 3–4 keep
# their original installed/read requirements above and below.
from candidate_evidence import candidate_rejections
from producer_timing import read_operations
dirty_tc = candidate_rejections(transactions, read_operations(root), initial_gen,
    first('publish dirty newer generation').get('at', start), removal.get('start', end))
write('candidate-rejection-join.json', dict(run_id=trace.get('run_id'),
    evidence_kind='pre-install only; no substitution for installed-verifier assertions',
    accepted_candidates=dirty_tc))
negative_preserved = bool(negative) and all('flower' in current_tc(r) and 'mirred' in current_tc(r) and 'serving 0' in r.get('readback', '') and member_up(r) is False for r in negative)
removed = current_tc(first('fresh removal read'))
final = first('final recovered state')
positive = first('final counted control').get('positive', {})
history = [e for r in clean for e in r['report'].get('errors', []) if e.get('stage') == 'tc_evidence']
state=read('recovery-applied-state.json', {})
model=(state.get('retained_applied') or {}).get('model', {})
# The agent replaces source=apply with the mandatory fresh source=live read
# before reporting. Prove completed Apply with its durable model, installed
# kernel generation and matching signed lease, not an impossible wire phase.
state_gen=str(model.get('generation', model.get('Generation', '0')))
state_apply=(int(state_gen)>initial_gen and state.get('applied_generation')==state_gen
             and (state.get('lease') or {}).get('generation')==state_gen
             and 'fvpn_generation='+state_gen in final.get('kernel_ruleset', '')
             and int((state.get('lease') or {}).get('member_count') or 0)>0
             and (state.get('member_path') or {}).get('generation')==state_gen)
capture_path=root/'redirect-node-capture.txt'
capture=capture_path.read_text() if capture_path.exists() else ''
packet_lines=[line for line in capture.splitlines() if re.match(r'^\d+\.\d+ ',line)]
reported=re.search(r'(?m)^(\d+) packets? captured$',capture)
capture_complete=bool(reported and int(reported[1])==len(packet_lines) and '0 packets dropped by kernel' in capture)
# Packet clock is UTC in tcpdump; the positive control interval is separately
# retained by the counted request. Keep both node boundaries in that interval.
from datetime import datetime
positive_start=datetime.fromisoformat(positive.get('at','1970-01-01T00:00:00Z').replace('Z','+00:00')).timestamp()
positive_end=datetime.fromisoformat(first('final counted control').get('at','1970-01-01T00:00:00Z').replace('Z','+00:00')).timestamp()
positive_packets=[line for line in packet_lines if positive_start<=float(line.split()[0])<=positive_end and '.443: Flags' in line]
packet_control=bool(any('fvmb0 In ' in line for line in positive_packets) and any('fvup0 Out ' in line for line in positive_packets))
checks = [
 dict(number=1, name='retained rejected read and quarantine', passed=bool(initial.get('member_forwarding_enabled') and tc(initial).get('verified') and rejections and any('serving 0' in r.get('readback', '') and member_up(r) is False and 'quarantine_fvmb0' in r.get('kernel_ruleset','') for r in labelled('containment wait')))),
 dict(number=2, name='all injected foreign objects removed with fresh read', passed=removal.get('ok') is True and bool(removed) and 'clsact' not in removed and 'flower' not in removed and 'mirred' not in removed),
 dict(number=3, name='clean newer generation and matching lease reach Apply', passed=bool(state_apply and matching)),
 dict(number=4, name='fresh verified Apply and live tc_evidence with canonical digest and historical rejection', passed=bool(state_apply and live and history) and all(digest(tc(r['report'])) for r in clean)),
 dict(number=5, name='restored topology marker serving and counted packet control', passed=bool(matching and positive.get('counted') and int(positive.get('admission_bytes_after') or 0)>int(positive.get('admission_bytes_before') or 0) and 'serving 1' in final.get('readback', '') and 'quarantine_fvmb0' not in final.get('kernel_ruleset', 'missing')) and 'kernel_ruleset' in final and member_up(final) is True and capture_complete and packet_control),
 dict(number=6, name='dirty newer generation remains non-serving and preserves foreign objects', passed=bool(dirty_tc and negative_preserved)),
]
write('recovery-verdict.json', dict(run_id=trace.get('run_id'), passed=all(c['passed'] for c in checks), status='pass' if all(c['passed'] for c in checks) else 'blocked', assertions=checks, initial_generation=str(initial_gen), durable_apply_generation=state_gen, durable_apply_verified=state_apply, node_any_capture_complete=capture_complete, recovered_packet_control=packet_control, apply_evidence='read-only retained_applied + member_path + installed generation + matching signed lease; qos source=apply is superseded by mandatory fresh live read before reporting', rejected_report_ids=[r['report']['report_id'] for r in rejections], apply_report_ids=[r['report']['report_id'] for r in apply], live_report_ids=[r['report']['report_id'] for r in live], matching_report_ids=[r['report']['report_id'] for r in matching], historical_error_rows=history, injected_creator='isolation harness explicitly installed clsact/flower/mirred', unsolicited_creator='No creator inferred from object kinds; see every retained rejected row in tc-evidence-index.json.'))
write('tc-evidence-index.json', dict(run_id=trace.get('run_id'), reads=[dict(report_id=r['report'].get('report_id'), observation_at=r['report'].get('observed_at'), status=r['status'], qos_detail=qos(r['report']), digest_matches=digest(tc(r['report'])), tc_evidence=tc(r['report'])) for r in reports if tc(r['report'])]))
# D-087: only real qualifying reports count, never force a fake-control switch on central.
qualifying=[]
for r in reports:
    b=r['report'];errors=b.get('errors') or []
    checks={c.get('name'):c for c in b.get('checks',[])}
    exact_checks=all(checks.get(name,{}).get('status')=='failed' and checks.get(name,{}).get('error_code')=='AWAITING_MEMBER_GRANTS' for name in ('lease','qos','source_binding'))
    if 200 <= r['status'] < 300 and (b.get('lease') or {}).get('lease_id') and b.get('provisioning_state')=='ready' and b.get('safety_status')=='passed' and b.get('member_forwarding_enabled') is False and b.get('forwarding_block_reasons')==['awaiting_member_grants'] and exact_checks and len(errors)==1 and errors[0].get('stage')=='lease' and errors[0].get('code')=='AWAITING_MEMBER_GRANTS' and errors[0].get('message')=='Serving lease has no member grants yet.':
        qualifying.append(r)
transition=read('member-grants-transition.json', {})
write('provisional-receiver-analysis.json', dict(run_id=receiver.get('run_id'), recipe='D-087 real lease-only eligibility switch on a clean fixed-generation published node', recipe_executed=transition.get('recipe_executed',False), transition='member-grants-transition.json', proof=transition.get('proof'), qualifying_reports=qualifying, observation_posts=len(reports)))
