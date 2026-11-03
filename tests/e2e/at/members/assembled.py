"""One foreground stack/run, with pre/post startup admission and teardown."""
import os
import subprocess
import sys
import json
import datetime
from pathlib import Path

sys.dont_write_bytecode = True
from admission import check, wait_ready


def main():
    wait = wait_ready()
    private = Path('.agent/tmp')
    private.mkdir(parents=True, exist_ok=True)
    before = datetime.datetime.now(datetime.timezone.utc).isoformat()
    vmstat = subprocess.check_output(['vmstat', '5', '7'], text=True)
    processes = subprocess.run(['pgrep', '-af', 'gates.sh|mk/at-'], capture_output=True, text=True)
    (private/'f16-preflight-pgrep.txt').write_text(processes.stdout)
    cache = subprocess.check_output(['go','env','GOCACHE'],text=True).strip()
    (private/'f16-gocache.txt').write_text(cache+'\n')
    (private/'f16-gocache.txt').chmod(0o600)
    home_prefix = str(Path.home()) + '/'
    cache_display = '${HOME}/' + cache[len(home_prefix):] if cache.startswith(home_prefix) else cache
    receipt = dict(at=before, baseline_commit=subprocess.check_output(['git','merge-base','HEAD','main'],text=True).strip(), main_ref_at_preflight=subprocess.check_output(['git','rev-parse','main'],text=True).strip(), harness_commit=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(), gocache=cache_display, gocache_note='Home directory redacted for publication; exact output retained privately, no cache override', vmstat=vmstat, suite_wait=wait, commands=['vmstat 5 7', "pgrep -af 'gates.sh|mk/at-'", 'go env GOCACHE'], pgrep='Original output retained privately; admission inventory above is sanitized')
    Path('docs/e2e/at/members/F16-host-preflight.json').write_text(json.dumps(receipt,indent=2)+'\n')
    if check():
        return 75
    env = dict(os.environ, FVPN_E2E_PREFIX='w4-members', FVPN_E2E_PORT_BASE='24000', FVPN_E2E_TWO_RESOLVERS='true', E2E_RESET='1', FVPN_E2E_KERNEL_OBSERVE='1')
    env.pop("FVPN_AT_MEMBERS_FOLLOWUP12", None)
    try:
        startup = subprocess.run(['make', '-f', 'mk/e2e.mk', 'e2e-up'], env=env).returncode
        if startup:
            return startup
        if check():
            return 75
        return subprocess.run(['make', '-f', 'mk/at-members.mk', 'at-members'], env=env).returncode
    finally:
        # Retain private original producer streams before the owned teardown.
        private = Path('.agent/tmp')
        private.mkdir(parents=True, exist_ok=True)
        for role, name in [('node', 'f16-parser-producer.jsonl'), ('node2', 'f16-node2-producer.jsonl')]:
            dest = private / name
            copied = subprocess.run(['docker', 'cp', 'fvpn-w4-members-' + role + ':/var/lib/vpn-node/e2e/enforcement-timing.jsonl', str(dest)], capture_output=True)
            if copied.returncode == 0:
                dest.chmod(0o600)
            print('Private producer export', role, 'exit:', copied.returncode, flush=True)
        down = subprocess.run(['make', '-f', 'mk/e2e.mk', 'e2e-down'], env=env).returncode
        print('Owned stack teardown exit:', down, flush=True)
        if down:
            raise RuntimeError('owned teardown failed')


if __name__ == '__main__':
    sys.exit(main())
