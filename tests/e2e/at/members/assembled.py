"""One foreground stack/run, with pre/post startup admission and teardown."""
import os
import subprocess
import sys

sys.dont_write_bytecode = True
from admission import check


def main():
    if check():
        return 75
    env = dict(os.environ, FVPN_E2E_PREFIX='w4-members', FVPN_E2E_PORT_BASE='24000', FVPN_E2E_TWO_RESOLVERS='true', E2E_RESET='1')
    env.pop("FVPN_AT_MEMBERS_FOLLOWUP12", None)
    try:
        startup = subprocess.run(['make', '-f', 'mk/e2e.mk', 'e2e-up'], env=env).returncode
        if startup:
            return startup
        if check():
            return 75
        return subprocess.run(['make', '-f', 'mk/at-members.mk', 'at-members'], env=env).returncode
    finally:
        down = subprocess.run(['make', '-f', 'mk/e2e.mk', 'e2e-down'], env=env).returncode
        print('Owned stack teardown exit:', down, flush=True)
        if down:
            raise RuntimeError('owned teardown failed')


if __name__ == '__main__':
    sys.exit(main())
