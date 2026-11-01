import sys,subprocess,time,json,hashlib,os
from pathlib import Path
root=Path.cwd(); evidence=root/'docs/poc/G2-followup23-evidence'; gates=evidence/'gates'; gates.mkdir(exist_ok=True)
cache=subprocess.check_output(['go','env','GOCACHE'],text=True).strip()
commands={
 'lane':['make','-f','mk/netlab.mk','netlab-lint','netlab-unit','netlab-build'],
 'consumers':['go','test','-count=1','./node/agent/...','./node/updater/...'],
 'hygiene':['go','test','-count=1','-v','./tests/netlab','-run','TestCommittedEvidenceFiles_ValidAndHygieneClean|TestD098_'],
 'handover':['go','test','-count=1','./tests/handover/...'],
 'go-lint':['make','go-lint'],
 'vet':['go','vet','./...'],
 'format':['sh','-ec','test -z "$(gofmt -l $(git ls-files \'*.go\'))"'],
 'diff':['git','diff','--check'],
}
for name in sys.argv[1:]:
 start=time.monotonic(); began=time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime())
 command=commands[name]
 with (root/'.agent'/('f23-gate-'+name+'.txt')).open('wb') as log:
  result=subprocess.run(['python3','tests/netlab/scripts/disktmp.py']+command,stdout=log,stderr=subprocess.STDOUT)
 raw=(root/'.agent'/('f23-gate-'+name+'.txt')).read_bytes()
 public=raw.decode().replace(str(root),'<worktree>').replace(cache,'<go-env-gocache>')
 (gates/(name+'.txt')).write_text(public)
 receipt=dict(command=command,started_utc=began,exit=result.returncode,elapsed_seconds=round(time.monotonic()-start,3),raw_sha256=hashlib.sha256(raw).hexdigest(),public_sha256=hashlib.sha256(public.encode()).hexdigest(),normalization='worktree and queried GOCACHE path replaced only',disk_temp='disktmp.py under .agent/tmp',gocache_queried=True)
 (gates/(name+'.json')).write_text(json.dumps(receipt,indent=2)+'\n')
 print(name,'exit',result.returncode,flush=True)
 print('\n'.join(public.splitlines()[-12:]),flush=True)
 if result.returncode: sys.exit(result.returncode)
