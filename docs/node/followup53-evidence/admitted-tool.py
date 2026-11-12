#!/usr/bin/env -S python3 -B
"""Keep admission messages separate from command stdout and stderr."""
import hashlib, json, os, pathlib, subprocess, sys, time
name = pathlib.Path(sys.argv[0]).name
env = dict(os.environ, CGO_ENABLED='0', GOTOOLCHAIN='auto', GOFLAGS='', FVPN_TEST_DB_SWEEP='0', PYTHONDONTWRITEBYTECODE='1')
env.pop('LD_LIBRARY_PATH', None)
real = {'make': env['F53_REAL_MAKE'], 'docker': env['F53_REAL_DOCKER'], 'node-go': env['F53_REAL_GO']}[name]
args = sys.argv[1:]
if name == 'node-go' and args and args[0] == 'test':
    args = ['test', '-json', *args[1:]]
log = pathlib.Path(env['F53_ADMISSION_LOG'])
start = time.monotonic()
while True:
    result = subprocess.run(['bash', env['FVPN_HOLD_SH']], env=env, capture_output=True, text=True)
    admitted = result.returncode == 0 and result.stdout.strip() == 'free'
    record = dict(tool=name, admitted=admitted, waited_seconds=round(time.monotonic()-start,3), argument_sha256=hashlib.sha256(json.dumps(args).encode()).hexdigest())
    if name == 'node-go':
        record['arguments'] = args
    with log.open('a') as f:
        f.write(json.dumps(record)+'\n')
    if admitted:
        break
    if time.monotonic()-start >= 600:
        print('admission unavailable; command not started', file=sys.stderr)
        sys.exit(125)
    time.sleep(10)
os.execve(real, [real, *args], env)
