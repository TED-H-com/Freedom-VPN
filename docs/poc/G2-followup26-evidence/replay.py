"""TEST ONLY pure offline replay: old assertions from task-start Git object."""
import ast
import gzip
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

sys.path.insert(0, str(Path('tests/netlab/scripts').resolve()))
from deadline_precision import check_precision

OUT = Path('docs/poc/G2-followup26-evidence')
BASE = json.loads((OUT / 'baseline.json').read_text())['head']
old = subprocess.check_output(['git', 'show', BASE+':tests/netlab/scripts/deadline.py'], text=True)
checks = [s for s in ast.parse(old).body if isinstance(s, ast.Assert) and s.lineno <= 35]
assert len(checks) == 2
old_checks = compile(ast.Module(body=checks, type_ignores=[]), '<original-deadline.py>', 'exec')


def before(values):
    try:
        exec(old_checks, {'values': values})
        return 'pass'
    except AssertionError:
        return 'fail'


def after(values):
    try:
        check_precision(values)
        return 'pass'
    except AssertionError:
        return 'fail'


def parse(text):
    units = {'d': 86400000, 'h': 3600000, 'm': 60000, 's': 1000, 'ms': 1}
    def duration(s):
        return sum(int(n)*units[u] for n,u in re.findall(r'(\d+)(ms|[dhms])',s))
    return [(duration(a),duration(b)) for a,b in re.findall(r'timeout (\w+) expires (\w+)',text)]


f16 = [(1001,699),(1002,700),(1003,701),(1004,702),(1005,703),(1006,704),(1007,706),(1008,707)]
rows = [dict(name='F16-01', provenance='Eight pairs supplied by orchestrator from isolation run 83b457ef; no original mechanism trace exists',
             values=f16, before=before(f16), after=after(f16))]
assert rows[0]['before'] == 'fail' and rows[0]['after'] == 'pass'
for r in json.loads((OUT/'historical-listings.json').read_text())['listings']:
    p = Path(r['path'])
    assert hashlib.sha256(p.read_bytes()).hexdigest() == r['sha256']
    values = parse(r['precision']['text'])
    assert [list(v) for v in values] == r['precision']['timeout_expires_ms']
    row = dict(name=r['path'], values=values, before=before(values), after=after(values))
    assert row['before'] == row['after'] == 'pass', row
    rows.append(row)
for quantum in (2,4,10,1000):
    values = [((1000+i)//quantum*quantum,(700+i)//quantum*quantum) for i in range(1,9)]
    # A complete text listing, not just a mutated verdict.
    listing = 'table inet fvpn {\n set fv_granularity {\n  type ipv4_addr\n  flags timeout\n  elements = { '+', '.join('192.0.2.%d timeout %dms expires %dms' % (i,*v) for i,v in enumerate(values,1))+' }\n }\n}\n'
    parsed = parse(listing)
    row = dict(name='coarse-'+str(quantum)+'ms', text=listing, values=parsed, before=before(parsed), after=after(parsed))
    assert row['before'] == row['after'] == 'fail', row
    rows.append(row)
result = dict(test_only=True, baseline=BASE, historical_listings=len(rows)-5, rows=rows)
(OUT/'replay.json').write_text(json.dumps(result,indent=2)+'\n')
print('F16-01: original assertion FAIL; replacement PASS')
print('Historical listings:',result['historical_listings'],'all before/after PASS')
print('Coarse 2/4/10/1000 ms text listings: before/after FAIL')
