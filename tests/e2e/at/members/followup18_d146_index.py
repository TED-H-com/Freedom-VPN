"""Index the original same-run D-146 verdicts without replacing any result."""
from collections import Counter
import hashlib
import json
from pathlib import Path

from evidence_io import load

ROOT = Path('docs/e2e/at/members')


def main():
    case = load(ROOT/'AT09.json')
    rows=[]
    for i, step in enumerate(case['steps']):
        if not step['name'].endswith(' D-146 clamp evidence'):
            continue
        value=json.loads(step['actual'])
        label=step['name'].removesuffix(' D-146 clamp evidence')
        transfer=next(s for s in case['steps'] if s['name']==label+' transfer')
        upload=next(s for s in case['steps'] if s['name']==label+' upload')
        original=json.loads(transfer['actual'])
        family=label.split()[1]
        path=Path('history/follow-up-18-captures')/case['run_id']/f'matrix-{family}-1280-d146'
        analysis=ROOT/path/'analysis.json'
        retained=json.loads(analysis.read_text())
        assert retained==value, 'inline and sidecar evidence differ'
        rows.append(dict(row=label,selected_family=original['selected_exit_before']['chosen_family'],
            original_transfer_status=transfer['status'],upload_status=upload['status'],evidence_status=step['status'],
            original_transfer=original['transfer'],qualified=value['qualified'],checks=value.get('checks'),reasons=value.get('reasons'),
            endpoint_mss={k:sorted({p['mss'] for p in v},key=lambda x:-1 if x is None else x) for k,v in value.get('endpoint_mss',{}).items()},
            reverse={k:v for k,v in value.get('reverse',{}).items() if k not in ('sender_packets','receiver_packets')},
            no_ptb=value.get('no_ptb'),input_applied_at=value.get('input_applied_at'),rules_ended_at=value.get('rules',{}).get('ended_at'),
            bulk_started_at=value.get('bulk_started_at'),bulk_ended_at=value.get('bulk_ended_at'),
            syn_times={k:[p['timestamp'] for p in v] for k,v in value.get('endpoint_mss',{}).items()},
            smaller_mss_control=value.get('smaller_mss_control'),artifact=str(path/'analysis.json'),sha256=hashlib.sha256(analysis.read_bytes()).hexdigest(),
            original_step_index=i))
    missing=[]
    for family in ('dual', 'ipv6', 'ipv4'):
        label=f'native {family} MTU 1280'
        if any(r['row']==label for r in rows):
            continue
        missing.append(dict(row=label,coverage='not exercised',
            original_steps=[s for s in case['steps'] if s['name'].startswith(label)],
            original_blockers=case.get('blockers', []),
            interpretation='No original D-146 qualification was produced. No selected family, handshake, bytes, MSS, outer-size or no-PTB result is inferred.'))
    result=dict(run_id=case['run_id'],source_commit=case['source_commit'],rows=rows,unexercised_rows=missing,
        interpretation='D-146 orchestrator reading: only successful unchanged TCP predicates with complete same-run clamp evidence avoid the D-137 downgrade. No historical status changed.',
        remaining_limits=['receive-only UDP and QUIC without PTB', 'MSS control below 1160 not exercised in unchanged native bulk fixture; netlab 34 stands',
            'clamp at 1280 on relay', 'real platforms', 'additional encapsulations or IPv6 extension-header budgets', 'IPv6 overlay (D-123 IPv6Block)', 'physical arm64'],
        historical_analysis='F18-historical-81b638e8-MSS.json')
    (ROOT/'F18-D146.json').write_text(json.dumps(result,indent=2)+'\n')
    lines=['# Follow-up 18 AT09 subchecks', '',f"Run `{case['run_id']}`, measured harness `{case['source_commit']}`. All original statuses below come from [AT09](AT09.json). Only D-146 qualifies the former receive-only TCP downgrade; no other predicate changes.", '',
           '| Index | Original check | Observed status |', '|---:|---|---|']
    for i,s in enumerate(case['steps']):
        lines.append(f"| {i} | {s['name'].replace('|','/')} | {s['status']} |")
    lines+=['','The original JSON retains each expected value, actual measurement, command and timestamp. [D-146 joins](F18-D146.json) and [restart rows](F18-AT09-restart.json) are additive indexes.', '',
            'Historical e80bb592, 0ffb355a and 81b638e8 outcomes stay unchanged. UDP/QUIC receive-only no-PTB remains a limitation; D-123, D-135 and D-136 remain in force.', '']
    (ROOT/'F18-AT09-subchecks.md').write_text('\n'.join(lines))
    print(json.dumps(dict(run_id=case['run_id'],rows=[{k:r[k] for k in ['row','selected_family','original_transfer_status','qualified','reasons']} for r in rows],AT09_subchecks=dict(Counter(s['status'] for s in case['steps'])))))


if __name__=='__main__':
    main()
