#!/usr/bin/env python3
"""D-140 schedule qualification; the unchanged Go evaluator owns the verdict."""
import json
import re
from evidence_io import read_json
from pathlib import Path
from review_unexecuted import unexecuted_expiry


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


def qualify_scope(rows, interleavings):
    by_id = {r['packet_id']:r for r in rows}
    # D-142 applies prospectively. Earlier archives retain their original verdict.
    positive = [r for r in rows if int(r['egress_mono_ns'])]
    drops = [r for r in rows if int(r['ingress_mono_ns']) and not int(r['egress_mono_ns'])]
    last = max(positive, key=lambda r:int(r['egress_mono_ns']), default=None)
    first = min(drops, key=lambda r:int(r['ingress_mono_ns']), default=None)
    start = min((rows.index(r) for r in (last, first) if r is not None), default=0)
    judged_ids = {r['packet_id'] for r in rows[start:]}
    for pair in interleavings:
        pair['d142_judged'] = pair['positive_packet'] in judged_ids
        pair['packets'] = [by_id[pair[k]] for k in ('positive_packet','next_packet')]
        pair['overlap_ns'] = str(int(pair['positive_egress_ns'])-int(pair['next_ingress_ns']))
    return start, last, first


# Validate before consuming fields. Historical complete plans remain readable;
# new diagnostic fields form one extension, required together when present.
DECIMAL = re.compile(r"-?(0|[1-9][0-9]*)\Z")


def decimal(value):
    return isinstance(value, str) and len(value) <= 20 and DECIMAL.fullmatch(value) is not None


SAMPLE = dict(packet_id=lambda v: type(v) is int and v > 0,
              **{k: decimal for k in ('send_mono_ns', 'ingress_mono_ns', 'egress_mono_ns', 'latency_ns', 'transit_ns')})


def fields(value, schema, path, errors):
    if not isinstance(value, dict):
        errors.append('malformed plan: '+path+' must be an object')
        return False
    valid = True
    for name, check in schema.items():
        if name not in value or not check(value[name]):
            errors.append('malformed plan: '+path+'.'+name+' missing or wrong type')
            valid = False
    return valid


def validate_plan(plan):
    errors = []
    if not isinstance(plan, dict):
        return {}, [], [], ['malformed plan: boundary schedule must be an object']
    schema = dict(qualified=lambda v: type(v) is bool,
                  samples=lambda v: isinstance(v, list),
                  planned_at=lambda v: isinstance(v, dict))
    schema.update({k: decimal for k in (
        'deadline_mono_ns', 'latency_min_ns', 'latency_median_ns', 'latency_max_ns',
        'latency_spread_ns', 'max_transit_ns', 'spacing_ns', 'predicted_first_ingress_ns',
        'predicted_last_ingress_ns', 'grid_anchor_offset_ns', 'original_boundary_predicted_ingress_offset_ns')})
    schema.update({k: lambda v: isinstance(v, str) for k in ('scope', 'decision_tradeoff', 'spacing_reason')})
    fields(plan, schema, 'plan', errors)
    fields(plan.get('planned_at'), dict(mono_ns=decimal, wall=lambda v: isinstance(v, str)), 'planned_at', errors)
    added = plan.get('added_scheduled_mono_ns')
    if 'added_scheduled_mono_ns' not in plan:
        errors.append('malformed plan: added_scheduled_mono_ns missing')
    elif added is None:
        errors.append('malformed plan: added_scheduled_mono_ns is null (F24-01 empty Go slice)')
    elif not isinstance(added, list):
        errors.append('malformed plan: added_scheduled_mono_ns must be a list')
    elif not added:
        errors.append('malformed plan: added_scheduled_mono_ns is empty')
    elif not all(decimal(v) for v in added):
        errors.append('malformed plan: added_scheduled_mono_ns entries must be decimal strings')
    samples = []
    if isinstance(plan.get('samples'), list):
        for i, sample in enumerate(plan['samples']):
            if fields(sample, SAMPLE, 'samples['+str(i)+']', errors):
                samples.append(sample)
    extension = dict(unqualified_reason=lambda v: isinstance(v, str),
                     discounted_samples=lambda v: isinstance(v, list),
                     spacing_unrounded_ns=decimal,
                     cadence_gaps_ns=lambda v: isinstance(v, list) and all(decimal(x) for x in v),
                     cadence_gap_rule=lambda v: isinstance(v, str),
                     candidates=lambda v: isinstance(v, list))
    if set(plan) & extension.keys():
        fields(plan, extension, 'plan', errors)
        if isinstance(plan.get('discounted_samples'), list):
            for i, sample in enumerate(plan['discounted_samples']):
                fields(sample, dict(SAMPLE, reason=lambda v: isinstance(v, str) and bool(v)), 'discounted_samples['+str(i)+']', errors)
        if isinstance(plan.get('candidates'), list):
            for i, candidate in enumerate(plan['candidates']):
                path = 'candidates['+str(i)+']'
                if fields(candidate, dict(scheduled_mono_ns=decimal, excluded_by_original_sends=lambda v: isinstance(v, list)), path, errors):
                    for j, send in enumerate(candidate['excluded_by_original_sends']):
                        fields(send, dict(scheduled_mono_ns=decimal, phase=lambda v: isinstance(v, str)), path+'.excluded_by_original_sends['+str(j)+']', errors)
    if plan.get('qualified') is False:
        # Keep the pre-existing check and wording; the explicit reason explains
        # calibrated-but-empty plans without treating calibration as enough.
        errors.append('same-run latency calibration unavailable')
        reason = plan.get('unqualified_reason')
        errors.append('unqualified plan: '+(reason if isinstance(reason, str) and reason else 'qualified is false'))
    return plan, samples, added if isinstance(added, list) and all(decimal(v) for v in added) else [], errors


def review(root):
    read=lambda name:read_json(root/name)
    current=read('AT13.json')
    blocked = unexecuted_expiry(root, 'boundary-review.json', current)
    if blocked:
        return blocked
    verdict=read('deadline-verdict.json')
    try:
        plan,samples,added,errors=validate_plan(read('boundary-schedule.json'))
    except (OSError, ValueError):
        plan,samples,added,errors={},[],[],['malformed plan: boundary-schedule.json missing or invalid JSON']
    rows,interleavings=timeline(verdict)
    by_id={r['packet_id']:r for r in rows}
    for s in samples:
        current=by_id.get(s['packet_id'],{})
        if any(current.get(k)!=s[k] for k in ('send_mono_ns','ingress_mono_ns','egress_mono_ns','latency_ns','transit_ns')):
            errors.append('calibration differs from final capture: '+str(s['packet_id']))
        if current.get('phase') not in ('phased','arbitrary_37ms'):errors.append('calibration not on original cadence')
        planned=plan.get('planned_at')
        if isinstance(planned,dict) and decimal(planned.get('mono_ns')) and int(s['egress_mono_ns'])>=int(planned['mono_ns']):errors.append('calibration was not measured before planning')
    start, last, first = qualify_scope(rows, interleavings)
    if any(p['d142_judged'] for p in interleavings):
        errors.append('positive transit interleaves next ingress within D-142 scope')
    # Exact original 101+101 schedules, without changing the evaluator or tests.
    d=int(verdict['deadline_mono_ns'])
    for phase,offset in [('phased',-101000000),('arbitrary_37ms',-64000000)]:
        actual=sorted(int(r['scheduled_mono_ns']) for r in rows if r['phase']==phase)
        expected=[d+offset+n*100000000 for n in range(-80,21)]
        if actual!=expected:errors.append('original cadence changed: '+phase)
    dense=[r for r in rows if r['phase']=='dense_boundary']
    if sorted(r['scheduled_mono_ns'] for r in dense)!=sorted(added) or not dense:
        errors.append('planned dense probes missing or extra')
    transit=[int(r['transit_ns']) for r in rows if r['transit_ns'] is not None]
    latency=[int(s['latency_ns']) for s in samples]
    result=dict(run_id=read('AT13.json')['run_id'],status='pass' if not errors and verdict['passed'] else 'fail',schedule_qualified=not errors,errors=errors,
        qualification_rule='D-142: every adjacent ingress-ordered pair from the earlier of the last positive and first drop through sweep end; earlier positive-positive overlaps recorded only',
        qualification_start_packet_id=rows[start]['packet_id'] if rows else None,
        deciding_pair=dict(last_positive=last,first_drop=first),
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
    r=review(Path(sys.argv[1]))
    if r['status']=='blocked':
        print('blocked: expiry not executed; refusing inherited boundary evidence')
    else:
        print(json.dumps({k:v for k,v in r.items() if k!='packets'},indent=2))
    raise SystemExit(0 if r['status']=='pass' else 1)
