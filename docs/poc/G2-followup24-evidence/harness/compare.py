#!/usr/bin/env python3
"""TEST ONLY D-133: same-host A,B,A,B, fresh lab each leg; no host networking.

Run from the worktree root after the required gates. All writes are follow-up 24
or ignored disk scratch. Quiet admission checks every measured vmstat interval; the first row is a
since-boot summary and is retained separately, as in the existing D-133 harness. Every rejected attempt is retained.
"""
import datetime, hashlib, json, math, os, pathlib, re, shutil, subprocess, tarfile, time
ROOT=pathlib.Path.cwd()
OUT=ROOT/'docs/poc/G2-followup24-evidence/comparison'
TMP=ROOT/'.agent/tmp/f24-comparison'
PREFIX='fvpn-w1-netlab-f24'
ENV=dict(os.environ, CGO_ENABLED='0', GOTOOLCHAIN='local', TMPDIR=str(TMP), GOTMPDIR=str(TMP))
BASE='335047a2b650db1d64bb4e0c59fc042fad74da51'
COUNT=60
START=None

def write(path, value):
 path.parent.mkdir(parents=True,exist_ok=True)
 path.write_text(json.dumps(value,indent=2)+'\n')

def run(args, **kw):
 return subprocess.run(args, env=kw.pop('env',ENV), capture_output=True, **kw)

def execute(name,args,check=True):
 p=run(args);raw=p.stdout+p.stderr
 public=raw.decode(errors='replace').replace(str(ROOT),'<worktree>')
 (OUT/(name+'.txt')).write_text(public)
 if check and p.returncode:raise RuntimeError(name+' failed; retained output')
 return p

def processes():
 p=run(['pgrep','-af','gates.sh|mk/at-|go test'],text=True)
 # Raw process command lines stay private; public receipt retains PID + kind.
 (TMP/'process-checks.txt').open('a').write(p.stdout)
 return [dict(pid=s.split()[0],kind='go test' if 'go test' in s else 'suite') for s in p.stdout.splitlines()]

def host(name):
 start=time.monotonic_ns();before=processes()
 load0=pathlib.Path('/proc/loadavg').read_text().split()[:3]
 def dstate():return sum(s.startswith('D') for s in run(['ps','-eLo','stat='],text=True).stdout.splitlines())
 d0=dstate();p=run(['vmstat','5','7'],text=True,check=True)
 rows=[]
 for line in p.stdout.splitlines():
  f=line.split()
  if len(f)>=17 and f[0].isdigit():rows.append(dict(r=int(f[0]),b=int(f[1]),id=int(f[14]),wa=int(f[15])))
 after=processes();intervals=rows[1:]
 quiet=len(intervals)==6 and not before and not after and all(r['r']<=3 and r['id']>=80 for r in intervals)
 row=dict(test_only=True,start_ns=str(start),end_ns=str(time.monotonic_ns()),load_before=load0,load_after=pathlib.Path('/proc/loadavg').read_text().split()[:3],d_state_before=d0,d_state_after=dstate(),processes_before=before,processes_after=after,vmstat=p.stdout,since_boot=rows[0],intervals=intervals,quiet=quiet)
 write(OUT/'host'/(name+'.json'),row)
 print('HOST',name,'quiet',quiet,'max_r',max(x['r'] for x in intervals),'min_idle',min(x['id'] for x in intervals),'suites',len(before),len(after),flush=True)
 return quiet

def await_quiet(name):
 attempt=max([int(p.stem.rsplit('-',1)[1]) for p in (OUT/'host').glob(name+'-*.json')] or [0])
 while time.monotonic()-START<90*60:
  attempt+=1;tick=time.monotonic()
  if host(name+'-'+str(attempt)) and time.monotonic()-START<90*60:return True
  time.sleep(max(0,60-(time.monotonic()-tick)))
 return False

def prepare():
 TMP.mkdir(parents=True,exist_ok=True);OUT.mkdir(parents=True,exist_ok=os.environ.get('FVPN_F24_RESUME')=='1')
 archive=TMP/'main.tar'
 with archive.open('wb') as f:subprocess.run(['git','archive',BASE],stdout=f,check=True)
 source=TMP/'main';source.mkdir()
 with tarfile.open(archive) as f:f.extractall(source,filter='data')
 archive.unlink()
 for label,cwd in [('A',source),('B',ROOT)]:
  p=run(['go','build','-buildvcs=false','-o',str(TMP/('binary-'+label)),'./tests/netlab/cmd/fvpn-netlab'],cwd=cwd)
  if p.returncode:raise RuntimeError(p.stderr.decode())
  p=run(['go','test','-c','-buildvcs=false','-tags','netlab','-o',str(TMP/('tests-'+label)),'./tests/netlab'],cwd=cwd)
  if p.returncode:raise RuntimeError(p.stderr.decode())
 write(OUT/('sources-resume.json' if os.environ.get('FVPN_F24_RESUME')=='1' else 'sources.json'),dict(test_only=True,A=BASE,B=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),binaries={kind+'-'+label:hashlib.sha256((TMP/(kind+'-'+label)).read_bytes()).hexdigest() for kind in ['binary','tests'] for label in ['A','B']},go_version=subprocess.check_output(['go','version'],text=True).strip(),cache_selected_by='go env GOCACHE',samples_per_leg=COUNT))
 shutil.rmtree(source)

def leg(name,label):
 attempt=0
 while time.monotonic()-START<90*60:
  attempt+=1;key=name+'-attempt'+str(attempt);dest=OUT/key;dest.mkdir(exist_ok=True)
  if (dest/'status.json').exists():raise RuntimeError('refusing to overwrite a completed attempt')
  if not await_quiet(key+'-prestart'):return None
  make=['make','-f','mk/netlab.mk','FVPN_LAB_PREFIX='+PREFIX]
  try:
   execute(key+'-up',make+['netlab-up'])
   execute(key+'-binary',['docker','cp',str(TMP/('binary-'+label)),PREFIX+':/usr/local/bin/fvpn-netlab'])
   if not await_quiet(key+'-ready'):return None
   fixture=json.loads((OUT.parent/'harness/comparison-model.json').read_text())['model']
   now=datetime.datetime.now(datetime.timezone.utc)
   stamp=lambda d:d.isoformat().replace('+00:00','Z')
   fixture['now']=stamp(now)
   for b in fixture['bindings']:
    b['valid_from']=stamp(now-datetime.timedelta(minutes=1));b['valid_until']=stamp(now+datetime.timedelta(hours=48))
   for lease in fixture['leases']:
    lease['signed_server_time']=stamp(now);lease['effective_service_end']=stamp(now+datetime.timedelta(hours=48));lease['node_lease_end']=stamp(now+datetime.timedelta(hours=48))
   write(dest/'input-model.json',dict(test_only=True,model=fixture))
   # Use plain production Apply for timed samples. One trace after them gives
   # the actual warm transaction bytes/rules without enabling sampling in them.
   records=[]
   for i in range(COUNT):
    m=dict(fixture,generation=42400+i)
    m['extra_denied_v4']=['192.0.2.200/32']+(['203.0.113.20/32'] if i%2 else [])
    p=run(['docker','exec','-i',PREFIX,'fvpn-netlab','apply','-deadline-window','120s'],input=json.dumps(m).encode())
    row=dict(index=i,exit=p.returncode)
    try:row['result']=json.loads(p.stdout)
    except ValueError:row['result_unavailable']=True
    if p.stderr:row['error']=p.stderr.decode(errors='replace')
    records.append(row)
    print('APPLY',key,i,'exit',p.returncode,flush=True)
   write(dest/'applies.json',dict(test_only=True,records=records))
   p=run(['docker','exec','-i',PREFIX,'fvpn-netlab','apply','-deadline-window','120s','-trace','/scratch/f24-shape.jsonl'],input=json.dumps(fixture).encode())
   write(dest/'shape-command.json',dict(test_only=True,exit=p.returncode,excluded_from_timing_population=True))
   shape=run(['docker','exec',PREFIX,'cat','/scratch/f24-shape.jsonl'],check=True).stdout
   (dest/'shape.jsonl').write_bytes(shape)
   # Fresh script is retained separately; warm shape above includes the guard.
   p=run(['docker','exec','-i',PREFIX,'fvpn-netlab','render'],input=json.dumps(fixture).encode(),check=True)
   (dest/'fresh-render.txt').write_bytes(p.stdout)
   for iface in ['fv-ih','fv-fh']:
    p=run(['docker','exec',PREFIX,'tc','-j','class','show','dev',iface],check=True)
    (dest/('tc-'+iface+'.txt')).write_bytes(p.stdout)
   testenv=dict(ENV,FVPN_NETLAB='1',FVPN_LAB_PREFIX=PREFIX,FVPN_NETLAB_EVIDENCE_DIR=str(dest/'reload'))
   p=run([str(TMP/('tests-'+label)),'-test.v','-test.timeout=10m','-test.run=^TestAT10_AT13_ReloadRevokeBaseline$'],cwd=ROOT/'tests/netlab',env=testenv)
   (dest/'reload-test.txt').write_text((p.stdout+p.stderr).decode(errors='replace').replace(str(ROOT),'<worktree>'))
   reload_exit=p.returncode
   print('RELOAD',key,'exit',reload_exit,flush=True)
   qualified=host(key+'-postflight')
   write(dest/'status.json',dict(test_only=True,leg=name,arm=label,postflight_quiet=qualified,reload_exit=reload_exit,failures=sum(r['exit']!=0 for r in records)))
   if qualified:return key
  finally:
   execute(key+'-down',make+['netlab-down'],check=False)
 return None

def pct(v):
 v=sorted(v)
 return {key:round(v[math.ceil(p*len(v))-1]/1e6,6) for key,p in [('p50',.5),('p90',.9),('p99',.99),('max',1)]} if v else None

def summarize(completed):
 rows=[];pooled={'A':[], 'B':[]}
 for name,key in completed.items():
  d=json.loads((OUT/key/'applies.json').read_text())['records'];pooled[name[0]]+=d
  shape=json.loads((OUT/key/'shape.jsonl').read_text().splitlines()[-1]);batch=next(b for b in shape['batches'] if b['kind']=='guard_install')
  reload_trace=OUT/key/'reload/TestAT10_AT13_ReloadRevokeBaseline/at10-apply.jsonl'
  reload_rows=[dict(index=i,exit=0 if r['timing']['succeeded'] else 1,result=r) for i,r in enumerate(json.loads(line) for line in reload_trace.read_text().splitlines())] if reload_trace.exists() else []
  status=json.loads((OUT/key/'status.json').read_text())
  fresh=(OUT/key/'fresh-render.txt').read_bytes()
  rows.append(dict(leg=name,attempt=key,reload_exit=status['reload_exit'],reload_timings=stats(reload_rows),fresh_rule_count=sum(l.startswith(b'add rule ') for l in fresh.splitlines()),fresh_script_bytes=len(fresh),guarded_rule_count=sum(l.startswith('add rule ') for l in batch['test_only_script'].splitlines()),guarded_script_bytes=len(batch['test_only_script'].encode()),script_sha256=batch['test_only_sha256'],**stats(d)))
 write(OUT/'summary.json',dict(test_only=True,legs=rows,pooled={k:stats(v) for k,v in pooled.items()},judging='D-138: all qualified legs pooled per arm; no two-leg envelope or attribution from one window. Failures and excluded attempts remain retained.'))

def stats(rows):
 guards=[];installs=[];verify=[];failed=[]
 for r in rows:
  t=r.get('result',{}).get('timing') or {};d=t.get('deadline',{})
  if d.get('guard_released'):
   guards.append(int(d['release_end_ns'])-int(d['install_start_ns']))
   installs.append(int(d['installed_by_ns'])-int(d['install_start_ns']))
  for v in t.get('verifications',[]):
   if v['verified']:verify.append(int(v['end_ns'])-int(v['start_ns']))
  if r['exit'] or not t.get('succeeded'):failed.append(r['index'])
 return dict(applies=len(rows),completed_guards=len(guards),failed_indices=failed,guard_ms=pct(guards),install_ms=pct(installs),verify_ms=pct(verify))

if __name__=='__main__':
 prepare();completed={}
 if (OUT/'window.json').exists():
  START=int(json.loads((OUT/'window.json').read_text())['start_ns'])/1e9
 else:
  START=time.monotonic()
  write(OUT/'window.json',dict(test_only=True,start_ns=str(time.monotonic_ns()),limit_seconds=5400,poll_seconds=60))
 try:
  for name,label in [('A1','A'),('B1','B'),('A2','A'),('B2','B')]:
   key=leg(name,label)
   if key is None:
    write(OUT/'blocked.json',dict(status='blocked: host not quiet',remaining_legs=[n for n in ['A1','B1','A2','B2'] if n not in completed],elapsed_seconds=time.monotonic()-START));break
   completed[name]=key
  summarize(completed)
 finally:
  execute('image-cleanup-resume' if os.environ.get('FVPN_F24_RESUME')=='1' else 'image-cleanup',['docker','image','rm',PREFIX+':lab'],check=False)
  shutil.rmtree(TMP)
