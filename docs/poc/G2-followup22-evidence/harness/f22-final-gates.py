from pathlib import Path
import datetime,json,subprocess,time
root=Path.cwd();logs=root/'.agent/f22-gates'
wrap=['python3','tests/netlab/scripts/disktmp.py']
gates=[
 ('agent-consumers',wrap+['go','test','-count=1','./node/agent/...','./node/updater/...']),
 ('committed-evidence',wrap+['go','test','-count=1','-v','./tests/netlab','-run','TestD098_|TestCommittedEvidenceFiles_ValidAndHygieneClean']),
 ('handover',wrap+['go','test','-count=1','./tests/handover/...']),
 ('go-lint',wrap+['make','go-lint','go-with-disk-tmp=']),
 ('vet',wrap+['go','vet','./...']),
 ('gofmt',['bash','-c','test -z "$(gofmt -l $(git ls-files \'*.go\'))"']),
 ('diff',['git','diff','--check'])]
results=[]
for name,args in gates:
 print('START',name,flush=True);start=time.monotonic()
 with (logs/(name+'.log')).open('w') as f:r=subprocess.run(args,stdout=f,stderr=subprocess.STDOUT)
 record=dict(gate=name,command=args,exit=r.returncode,seconds=round(time.monotonic()-start,3),completed=datetime.datetime.now(datetime.timezone.utc).isoformat())
 results.append(record);print('END',name,r.returncode,flush=True)
 (logs/'final-results.json').write_text(json.dumps(results,indent=2)+'\n')
raise SystemExit(int(any(r['exit'] for r in results)))
