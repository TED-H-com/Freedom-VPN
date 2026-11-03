"""TEST ONLY host driver; all netlink and nft operations stay in the lab."""
import collections
import gzip
import json
from pathlib import Path
import subprocess
import sys

OUT = Path('docs/poc/G2-followup26-evidence')
PREFIX = 'fvpn-w1-netlab-f26-mechanism'
mode = sys.argv[1]
path = OUT / ('mechanism-' + mode + '.jsonl.gz')
assert not path.exists()
result = subprocess.run(['docker', 'exec', PREFIX, 'python3', '-B',
                         '/work/docs/poc/G2-followup26-evidence/probe.py', mode], capture_output=True)
path.write_bytes(gzip.compress(result.stdout, mtime=0))
if result.returncode:
    (OUT / ('mechanism-' + mode + '.err')).write_bytes(result.stderr)
    print(result.stderr.decode(), file=sys.stderr)
    sys.exit(result.returncode)
rows = [json.loads(line) for line in result.stdout.splitlines()]
if mode in ('normal', 'padded'):
    trials = rows[1:]
    summary = dict(platform=rows[0], trials=len(trials),
                   old_failures=[r['index'] for r in trials if not r['old_pass']],
                   allocation_split_proven=[r['index'] for r in trials if r['allocation_split_proven']])
else:
    vectors = []
    for r in rows[1:]:
        ages = [(int(a.rsplit('.', 1)[1]), 700+int(a.rsplit('.', 1)[1])-e) for a, _, e in r['dump']['wire_order']]
        v = [e for _, e in sorted(ages)]
        vectors.append(tuple(e-v[0] for e in v))
    summary = dict(platform=rows[0], dumps=len(vectors),
                   relative_age_vectors=[dict(vector=v, count=n) for v,n in collections.Counter(vectors).most_common()])
(OUT / ('mechanism-' + mode + '-summary.json')).write_text(json.dumps(summary, indent=2)+'\n')
print(json.dumps(summary, indent=2))
