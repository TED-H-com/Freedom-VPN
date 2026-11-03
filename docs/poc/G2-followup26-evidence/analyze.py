"""TEST ONLY derive witnesses and rates from the retained netlink experiments."""
import collections
import gzip
import json
from pathlib import Path
import sys

sys.path.insert(0, str(Path('tests/netlab/scripts').resolve()))
from deadline_precision import check_precision

OUT = Path('docs/poc/G2-followup26-evidence')


def read(mode):
    return [json.loads(line) for line in gzip.decompress((OUT/('mechanism-'+mode+'.jsonl.gz')).read_bytes()).splitlines()]


def ages(dump):
    return [(int(a.rsplit('.',1)[1]), 700+int(a.rsplit('.',1)[1])-e) for a,t,e in dump['wire_order']]


def relative(dump):
    v = [age for _,age in sorted(ages(dump))]
    return tuple(a-v[0] for a in v)


def decreases(dump):
    v = ages(dump)
    return [dict(earlier_key=a[0],later_key=b[0],earlier_age_ms=a[1],later_age_ms=b[1],
                 allocation_base_separation_at_least_ms=a[1]-b[1])
            for a,b in zip(v,v[1:]) if b[1] < a[1]]


normal = read('normal')[1:]
assert len(normal) == 1000
steps = collections.Counter()
allocations, changed_reads, old_failures, f16_shape = [],[],[],[]
for trial in normal:
    p = trial['precision']['timeout_expires_ms']
    check_precision(p)
    steps.update(p[i+1][1]-p[i][1] for i in range(7))
    if not trial['old_pass']:
        old_failures.append(trial['index'])
    age = [701+i-e for i,(_,e) in enumerate(p)]
    if [a-min(age) for a in age] == [1,1,1,1,1,1,0,0]:
        f16_shape.append(trial['index'])
    witnesses = [(j,decreases(d)) for j,d in enumerate(trial['dumps']) if decreases(d)]
    if witnesses:
        allocations.append(dict(trial=trial['index'],dump=witnesses[0][0],decreases=witnesses[0][1]))
    if len({relative(d) for d in trial['dumps']}) > 1:
        changed_reads.append(trial['index'])

padded = []
for trial in read('padded')[1:]:
    check_precision(trial['precision']['timeout_expires_ms'])
    padded.append(dict(direction=trial['install']['padded'],
                       timeout_expires_ms=trial['precision']['timeout_expires_ms'],
                       relative_age_vectors=[list(relative(d)) for d in trial['dumps']],
                       decreases_by_dump=[decreases(d) for d in trial['dumps']]))

dumps = read('dump')[1:]
assert len(dumps) == 2000
vectors = [relative(r['dump']) for r in dumps]
changes = [i for i in range(1,len(vectors)) if vectors[i] != vectors[i-1]]
result = dict(test_only=True, normal=dict(trials=len(normal),old_failures=old_failures,
              old_failure_percent=100*len(old_failures)/len(normal),new_failures=0,
              f16_age_shape_trials=f16_shape,adjacent_expires_steps=dict(sorted(steps.items())),
              allocation_split_witnesses=allocations,allocation_split_count=len(allocations),
              allocation_split_percent_lower_bound=100*len(allocations)/len(normal),
              within_trial_dump_clock_change=changed_reads),
              padded=padded,
              unchanged_set=dict(dumps=len(dumps),relative_age_vectors=[dict(vector=v,count=n) for v,n in collections.Counter(vectors).most_common()],
                 transitions=changes,first_transition=None if not changes else dumps[max(0,changes[0]-1):changes[0]+2]),
              limit='Allocation count is a witnessed lower bound, not an estimate for all schedules. Sorted F16-01 alone cannot distinguish the two clock mechanisms.')
(OUT/'mechanism-analysis.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(dict(trials=len(normal),old_failures=len(old_failures),new_failures=0,
                     allocation_split_witnesses=len(allocations),f16_shape=len(f16_shape),
                     dump_only_vector_count=len(set(vectors)),dump_only_transitions=len(changes)),indent=2))
