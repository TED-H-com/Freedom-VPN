import datetime, hashlib, json, math, os, pathlib, shutil, subprocess, sys, tarfile, time
ROOT=pathlib.Path.cwd()
BASE=ROOT/'.agent/f22-runtime'
BASE.mkdir(exist_ok=True)
PREFIX='fvpn-w1-netlab-f22'
ENV=dict(os.environ,CGO_ENABLED='0',GOTOOLCHAIN='local',GOMAXPROCS='4',GOFLAGS='-p=4',FVPN_LAB_PREFIX=PREFIX)
TMP=ROOT/'.agent/tmp/f22-run'
TMP.mkdir(exist_ok=True)
ENV.update(TMPDIR=str(TMP),GOTMPDIR=str(TMP))

def write(p,x):
 p.parent.mkdir(parents=True,exist_ok=True)
 p.write_text(json.dumps(x,indent=2)+'\n')

def run(name,args,cwd=ROOT,env=None,check=True):
 start=time.monotonic_ns()
 path=BASE/(name+'.log');path.parent.mkdir(parents=True,exist_ok=True)
 print('START',name,flush=True)
 with path.open('w') as f: p=subprocess.run(args,cwd=cwd,env=env or ENV,stdout=f,stderr=subprocess.STDOUT)
 row=dict(name=name,args=args,exit=p.returncode,start_ns=str(start),end_ns=str(time.monotonic_ns()))
 with (BASE/'gates.jsonl').open('a') as f:f.write(json.dumps(row)+'\n')
 print('END',name,'exit',p.returncode,flush=True)
 if check and p.returncode: raise RuntimeError(name+' failed')
 return p.returncode

def output(args):return subprocess.check_output(args,text=True).strip()

def suites():
 p=subprocess.run(['pgrep','-af','gates.sh|mk/at-|go test'],text=True,capture_output=True)
 return p.stdout.strip()

def host(name):
 start=time.monotonic_ns();before=suites()
 load_before=pathlib.Path('/proc/loadavg').read_text().strip()
 d_before=sum(x.startswith('D') for x in output(['ps','-eLo','stat=']).splitlines())
 p=subprocess.run(['vmstat','5','7'],text=True,capture_output=True,check=True)
 rows=[]
 for line in p.stdout.splitlines():
  f=line.split()
  if len(f)>=17 and f[0].isdigit(): rows.append(dict(r=int(f[0]),b=int(f[1]),id=int(f[14]),wa=int(f[15])))
 intervals=rows[1:]
 after=suites()
 quiet=len(intervals)==6 and not before and not after and all(r['r']<=3 and r['id']>=80 for r in intervals)
 row=dict(test_only=True,start_ns=str(start),end_ns=str(time.monotonic_ns()),load_before=load_before,load_after=pathlib.Path('/proc/loadavg').read_text().strip(),d_state_before=d_before,d_state_after=sum(x.startswith('D') for x in output(['ps','-eLo','stat=']).splitlines()),suite_processes_before=before,suite_processes_after=after,vmstat=p.stdout,since_boot=rows[0],intervals=intervals,quiet=quiet)
 write(BASE/(name+'.json'),row)
 print('HOST',name,'quiet',quiet,'r',max(r['r'] for r in intervals),'idle',min(r['id'] for r in intervals),'suites',bool(before or after),flush=True)
 return quiet

WAIT_START=None

def await_quiet(name):
 global WAIT_START
 if WAIT_START is None:
  prior=[json.loads(p.read_text())['start_ns'] for p in BASE.glob('*preflight-*.json')]
  WAIT_START=min(map(int,prior))/1e9 if prior else time.monotonic()
 attempt=max([int(p.stem.rsplit('-',1)[1]) for p in BASE.glob(name+'-*.json') if p.stem.rsplit('-',1)[1].isdigit()] or [0])
 while time.monotonic()-WAIT_START<90*60:
  attempt+=1;start=time.monotonic()
  if host(name+'-'+str(attempt)): return
  time.sleep(max(0,60-(time.monotonic()-start)))
 write(BASE/'blocked.json',dict(status='blocked: host not quiet',seconds=time.monotonic()-WAIT_START))
 raise RuntimeError('blocked: host not quiet')


def prepare(label,source):
 dest=BASE/('source-'+label)
 if label=='A':
  dest.mkdir(exist_ok=True)
  tar=TMP/'main.tar'
  with tar.open('wb') as f:subprocess.run(['git','archive',source],cwd=ROOT,stdout=f,check=True)
  with tarfile.open(tar) as f:f.extractall(dest,filter='data')
  tar.unlink()
 else:dest=ROOT
 binaries=BASE/('bin-'+label);binaries.mkdir(exist_ok=True)
 for binary,pkg in [('fvpn-netlab','./tests/netlab/cmd/fvpn-netlab'),('fvpn-agent','./node/agent/cmd/fvpn-agent'),('fvpn-watchdog','./node/watchdog/cmd/fvpn-watchdog'),('fvpn-labcontrol','./node/internal/fakecontrol/cmd/fvpn-labcontrol')]:
  run('build-'+label+'-'+binary,['go','build','-o',str(binaries/binary),pkg],cwd=dest)
 image=PREFIX+'-'+label.lower()+':lab'
 run('image-'+label,['docker','build','-t',image,str(dest/'tests/netlab')])
 write(BASE/('source-'+label+'.json'),dict(test_only=True,commit=source,binaries={p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in binaries.iterdir()}))
 return dest,binaries,image


def suite_once(leg,label,config,which):
 dest,binaries,image=config
 await_quiet(leg+'-'+which+'-preflight')
 name=leg+'-'+which
 out=BASE/name;out.mkdir(exist_ok=True)
 make=['make','-f','mk/netlab.mk','FVPN_LAB_PREFIX='+PREFIX,'NETLAB_IMAGE='+image]
 env=dict(ENV,FVPN_NETLAB='1',FVPN_NETLAB_EVIDENCE_DIR=str(out/'live'),FVPN_AGENT_EVIDENCE_DIR=str(out/'agent'))
 try:
  run(name+'-up',make+['netlab-up'],cwd=dest)
  active=suites();write(out/'admission-after-start.json',dict(processes=active))
  if active:raise RuntimeError('new concurrent suite at startup')
  run(name+'-collector-dir',['docker','exec',PREFIX,'mkdir','-p','/usr/local/lib/f22'])
  for binary in binaries.iterdir():
   target='/usr/local/lib/f22/fvpn-netlab' if binary.name=='fvpn-netlab' else '/usr/local/bin/'+binary.name
   run(name+'-copy-'+binary.name,['docker','cp',str(binary),PREFIX+':'+target])
  run(name+'-collector',['docker','cp',str(ROOT/'tests/netlab/scripts/f22_command.py'),PREFIX+':/usr/local/bin/fvpn-netlab'])
  run(name+'-chmod',['docker','exec',PREFIX,'chmod','755','/usr/local/bin/fvpn-netlab'])
  # Startup/build work is excluded from qualification; recheck before traffic.
  await_quiet(name+'-ready')
  if which=='live':
   args=['go','test','-tags','netlab','-count=1','-timeout','12m','-p','1','-parallel','1','-v','./tests/netlab','-run','^Test(AT|D104|Docker|Foreign|ISO|Ingress|NetBird|Probe|Revoke)']
  elif which in ['agent','agent-trace']:
   if which=='agent-trace':env['FVPN_D116_MARKER']='1'
   args=['go','test','-tags','netlab','-count=1','-timeout','12m','-p','1','-parallel','1','-v','./tests/netlab/agent']
   if which=='agent-trace':args+=['-run','^TestD116_AT10_InstallerMarkerEnrollment$']
  else:
   env['FVPN_NETLAB_F22']='1'
   args=['go','test','-tags','netlab','-count=1','-timeout','4m','-p','1','-parallel','1','-v','./tests/netlab','-run','^TestF22_']
  code=run(name+'-suite',args,cwd=dest,env=env,check=False)
  run(name+'-commands',['docker','cp',PREFIX+':/scratch/f22-command-results',str(out/'commands')],check=False)
  post=host(name+'-postflight')
  write(out/'status.json',dict(exit=code,postflight_quiet=post,source=label))
  if which=='diagnostic':
   private=ROOT/'.agent/f22-private';private.mkdir(exist_ok=True)
   run(name+'-pcaps',['docker','cp',PREFIX+':/scratch/f22-rollback',str(private)],check=False)
 finally:
  run(name+'-down',make+['netlab-down'],cwd=dest,check=False)
 if code!=0: print('RETAINED SUITE FAILURE',name,flush=True)
 return post


def archive_unqualified(name):
 parent=BASE/'unqualified';parent.mkdir(exist_ok=True)
 attempt=1
 while (parent/(name+'-'+str(attempt))).exists():attempt+=1
 archive=parent/(name+'-'+str(attempt));archive.mkdir()
 for p in list(BASE.iterdir()):
  if p.name==name or (p.name.startswith(name+'-') and not p.name.startswith(name+'-trace-')):
   shutil.move(str(p),str(archive/p.name))
 print('RETAINED UNQUALIFIED POSTFLIGHT',name,str(attempt),flush=True)

def suite(leg,label,config,which):
 name=leg+'-'+which
 status=BASE/name/'status.json'
 if status.exists():
  if json.loads(status.read_text())['postflight_quiet']:
   print('KEEP QUALIFIED',name,flush=True);return
  archive_unqualified(name)
 while not suite_once(leg,label,config,which):
  archive_unqualified(name)

if __name__=='__main__':
 configs={}
 try:
  commits={'A':json.loads((BASE/'source-A.json').read_text())['commit'] if (BASE/'source-A.json').exists() else output(['git','rev-parse','main']),'B':output(['git','rev-parse','HEAD'])}
  write(BASE/'comparison.json',dict(test_only=True,order=['A1','B1','A2','B2'],commits=commits,scope='live suite opts command measurement through TEST ONLY trace; ordinary agent suite measures disabled path (agent opt-in is an owner hand-off); common post-exit Apply collector on both sources'))
  for label in ['A','B']:
   if (BASE/('source-'+label+'.json')).exists():
    assert json.loads((BASE/('source-'+label+'.json')).read_text())['commit']==commits[label], 'source changed: use a fresh evidence directory'
    dest=BASE/'source-A' if label=='A' else ROOT
    image=PREFIX+'-'+label.lower()+':lab'
    run('image-resume-'+label,['docker','build','-t',image,str(dest/'tests/netlab')])
    configs[label]=(dest,BASE/('bin-'+label),image)
   else:configs[label]=prepare(label,commits[label])
  for leg in ['A1','B1','A2','B2']:
   for which in (['live','agent','agent-trace'] if leg in ['A1','B1'] else ['live','agent']):
    suite(leg,leg[0],configs[leg[0]],which)
    if leg=='B1' and which=='live':suite('B1','B',configs['B'],'diagnostic')
 finally:
  for label in ['a','b']:
   run('remove-image-'+label,['docker','image','rm',PREFIX+'-'+label+':lab'],check=False)
  # netlab-up also builds a binary per source; remove only this run's owned files.
  for source in [ROOT,BASE/'source-A']:
   p=source/'.agent/tmp'/ (PREFIX+'-fvpn-netlab')
   if p.exists():p.unlink()
  write(BASE/'cleanup.json',dict(containers=output(['docker','ps','-a','--filter','name='+PREFIX,'--format','{{.Names}}']),images=output(['docker','images','--filter','reference='+PREFIX+'*','--format','{{.Repository}}:{{.Tag}}']),networks=output(['docker','network','ls','--filter','name='+PREFIX,'--format','{{.Name}}']),volumes=output(['docker','volume','ls','--filter','name='+PREFIX,'--format','{{.Name}}'])))
