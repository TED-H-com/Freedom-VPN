import json, os, pathlib, re, signal, subprocess, sys, time

if sys.argv[1] == '--worker':
    end = time.monotonic() + float(sys.argv[2])
    x = 1
    while time.monotonic() < end:
        for _ in range(10000):
            x = (x * 1664525 + 1013904223) & 0xffffffff
    sys.exit(0)
name, load = sys.argv[1], int(sys.argv[2])
cmd = sys.argv[3:]
root = pathlib.Path.cwd()
out = root / '.agent/f53-fix1'
out.mkdir(parents=True, exist_ok=True)
if (out/(name+'.json')).exists() or (out/(name+'.log')).exists():
    raise SystemExit('refusing to replace a recorded attempt')
env = dict(os.environ, CGO_ENABLED='0', GOTOOLCHAIN='local', GOFLAGS='', PYTHONDONTWRITEBYTECODE='1')
env.pop('LD_LIBRARY_PATH', None)
if name.startswith('gate-'):
    import shutil
    env.update(F53_REAL_MAKE=shutil.which('make'), F53_REAL_DOCKER=shutil.which('docker'), F53_REAL_GO=shutil.which('go'), F53_REAL_PYTHON=sys.executable,
               F53_ADMISSION_LOG=str(out/'child-admission.jsonl'), FVPN_TEST_DB_SWEEP='0')
    tool_dir = pathlib.Path(env['TMPDIR'])/'f53-tools'
    tool_dir.mkdir(exist_ok=True)
    wrapper = pathlib.Path(__file__).with_name('admitted-tool.py')
    for tool in ('make', 'docker', 'node-go'):
        shutil.copyfile(wrapper, tool_dir/tool)
        (tool_dir/tool).chmod(0o755)
    python_wrapper = tool_dir/'python3'
    python_wrapper.write_text('#!/bin/sh\nexec "$F53_REAL_PYTHON" -B "$@"\n')
    python_wrapper.chmod(0o755)
    env['PATH'] = str(tool_dir) + os.pathsep + env['PATH']
assert env['TMPDIR'] == env['GOTMPDIR']
fs = subprocess.check_output(['findmnt','-T',env['TMPDIR'],'-n','-o','FSTYPE'],text=True).strip()
assert fs not in ('tmpfs','ramfs'), fs
admit = subprocess.run(['bash',env['FVPN_HOLD_SH']],env=env,capture_output=True,text=True)
print(admit.stdout, end='', flush=True)
if admit.returncode or admit.stdout.strip() != 'free':
    sys.exit(125)
ps = subprocess.check_output(['ps','-eo','pid,etimes,args'],text=True)
pattern = r'[g]ates\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test|[e]2e-test'
ancestors=set()
pid=os.getpid()
while pid:
    ancestors.add(str(pid))
    try:
        pid=int(pathlib.Path('/proc',str(pid),'stat').read_text().split(') ',1)[1].split()[1])
    except (FileNotFoundError,ValueError):
        break
conflicts = [line.split(None,2)[0] for line in ps.splitlines() if re.search(pattern,line) and line.split(None,2)[0] not in ancestors]
if conflicts:
    print('suite overlap pids:', ','.join(conflicts), flush=True)
    sys.exit(125)
head = subprocess.check_output(['git','rev-parse','HEAD'],env=env,text=True).strip()
diff = subprocess.check_output(['git','diff','--binary','HEAD'],env=env)
import hashlib
(out/(name+'.patch')).write_bytes(diff)
meta = dict(command=cmd, head=head, overlay_sha256=hashlib.sha256(diff).hexdigest(), load_workers=load, scratch_fstype=fs, admission='free', overlap_pids=conflicts)
files=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','node','tests/netlab/agent','mk/node.mk','go.mod','go.sum'],env=env,text=True).splitlines()
meta['source_sha256']={f:hashlib.sha256(pathlib.Path(f).read_bytes()).hexdigest() for f in sorted(set(files)) if pathlib.Path(f).is_file()}
workers=[]
command_process=None
started=time.monotonic()
try:
    for _ in range(load):
        workers.append(subprocess.Popen([sys.executable,'-B',__file__,'--worker','3600'],env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL))
    with (out/(name+'.log')).open('w') as log:
        command_process = subprocess.Popen(cmd,env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
        command_process.wait(timeout=3500)
    meta['exit_status'] = command_process.returncode
except subprocess.TimeoutExpired:
    meta['supervisor_timeout'] = True
    meta['exit_status'] = 124
finally:
    if command_process is not None and command_process.poll() is None:
        os.killpg(command_process.pid,signal.SIGTERM)
        try:
            command_process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(command_process.pid,signal.SIGKILL)
            command_process.wait()
    for worker in workers:
        if worker.poll() is None:
            worker.terminate()
    for worker in workers:
        worker.wait()
    meta['workers_joined']=len(workers)
    meta['elapsed_seconds']=round(time.monotonic()-started,3)
    (out/(name+'.json')).write_text(json.dumps(meta,indent=2)+'\n')
print(json.dumps({k:v for k,v in meta.items() if k!="source_sha256"}),flush=True)
print('\n'.join((out/(name+'.log')).read_text().splitlines()[-12:]),flush=True)
sys.exit(meta['exit_status'])
