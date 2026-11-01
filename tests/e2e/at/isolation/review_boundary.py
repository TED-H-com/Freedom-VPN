#!/usr/bin/env python3
"""D-140 schedule qualification; the unchanged Go evaluator owns the verdict."""
import json
from evidence_io import read_json
from pathlib import Path


def timeline(verdict):
    deadline=int(verdict['deadline_mono_ns'])
    rows=[]
    for r in verdict['packets']:
        p=r['probe']; send=int(p['send']['mono_ns']); scheduled=int(p['scheduled_mono_ns'])
        ingress=int(r['ingress_mono_ns']);egress=int(r['egress_mono_ns'])
        rows.append(dict(packet_id=p['packet_id'],phase=p['phase'],result=p['result'],
            scheduled_mono_ns=str(scheduled),send_mono_ns=str(send),ingress_mono_ns=str(ingress),egress_mono_ns=str(egress),
            scheduled_offset_ns=str(scheduled-deadline),send_offset_ns=str(send-deadline),
            ingress_offset_ns=str(ingress-deadline) if ingress else None,egress_offset_ns=str(egress-deadline) if egress else None,
            scheduling_lateness_ns=str(send-scheduled),latency_ns=str(ingress-send) if ingress else None,
            transit_ns=str(egress-ingress) if egress and ingress else None))
    rows.sort(key=lambda p:int(p['ingress_mono_ns']))
    interleavings=[]
    for left,right in zip(rows,rows[1:]):
        if int(left['egress_mono_ns']) and int(right['ingress_mono_ns']) and int(left['egress_mono_ns'])>=int(right['ingress_mono_ns']):
            interleavings.append(dict(positive_packet=left['packet_id'],next_packet=right['packet_id'],
                positive_egress_ns=left['egress_mono_ns'],next_ingress_ns=right['ingress_mono_ns']))
    return rows,interleavings


def review(root):
    read=lambda name:read_json(root/name)
    current=read('AT13.json')
    expiry=next((c for c in current.get('checks',[]) if c['name']=='short whole-node lease expiry'),{})
    if 'D-104 exact packet deadline verdict=' not in expiry.get('actual',''):
        raise ValueError('Current attempt did not execute expiry; refusing inherited boundary evidence')
    verdict=read('deadline-verdict.json');plan=read('boundary-schedule.json')
    rows,interleavings=timeline(verdict);errors=[]
    if not plan['qualified']:errors.append('same-run latency calibration unavailable')
    by_id={r['packet_id']:r for r in rows}
    samples=plan.get('samples') or []
    for s in samples:
        current=by_id.get(s['packet_id'],{})
        if any(current.get(k)!=s[k] for k in ('send_mono_ns','ingress_mono_ns','egress_mono_ns','latency_ns','transit_ns')):
            errors.append('calibration differs from final capture: '+str(s['packet_id']))
        if current.get('phase') not in ('phased','arbitrary_37ms'):errors.append('calibration not on original cadence')
        if int(s['egress_mono_ns'])>=int(plan['planned_at']['mono_ns']):errors.append('calibration was not measured before planning')
    if interleavings:errors.append('positive transit interleaves next ingress')
    # Exact original 101+101 schedules, without changing the evaluator or tests.
    d=int(verdict['deadline_mono_ns'])
    for phase,offset in [('phased',-101000000),('arbitrary_37ms',-64000000)]:
        actual=sorted(int(r['scheduled_mono_ns']) for r in rows if r['phase']==phase)
        expected=[d+offset+n*100000000 for n in range(-80,21)]
        if actual!=expected:errors.append('original cadence changed: '+phase)
    dense=[r for r in rows if r['phase']=='dense_boundary']
    if sorted(r['scheduled_mono_ns'] for r in dense)!=sorted(plan.get('added_scheduled_mono_ns',[])) or not dense:
        errors.append('planned dense probes missing or extra')
    transit=[int(r['transit_ns']) for r in rows if r['transit_ns'] is not None]
    latency=[int(s['latency_ns']) for s in samples]
    result=dict(run_id=read('AT13.json')['run_id'],schedule_qualified=not errors,errors=errors,
        unchanged_evaluator_passed=verdict['passed'],unchanged_evaluator_errors=verdict['errors'],
        last_positive_offset_ns=verdict['last_positive_offset_ns'],first_drop_offset_ns=verdict['first_drop_offset_ns'],
        calibration_count=len(samples),calibration_latency_min_ns=str(min(latency)) if latency else None,
        calibration_latency_max_ns=str(max(latency)) if latency else None,
        calibration_latency_spread_ns=str(max(latency)-min(latency)) if latency else None,
        spacing_ns=plan.get('spacing_ns'),dense_probes=len(dense),all_probes=len(rows),
        maximum_observed_transit_ns=str(max(transit)) if transit else None,interleavings=interleavings,packets=rows,
        scope='All attempts counted. Capture-read timestamps are userspace bounds, not exact kernel instants. Qualification does not replace EvaluateDeadline or promote a failure. Original phased and arbitrary_37ms schedules are checked exactly. No node rules added.',
        historical_isolation_F14_01='history/follow-up-14/deadline-verdict.json: run 56d20530 remains failed; distinct from members F14-01; four renewal losses 15–18 outside guards remain counted.')
    (root/'boundary-review.json').write_text(json.dumps(result,indent=2)+'\n')
    return result


if __name__=='__main__':
    import sys
    r=review(Path(sys.argv[1]));print(json.dumps({k:v for k,v in r.items() if k!='packets'},indent=2))
    raise SystemExit(0 if r['schedule_qualified'] else 1)
