"""TEST ONLY sole resumed campaign, launched after three quiet preflights."""
from pathlib import Path
import subprocess
import sys

PREFIX = 'fvpn-w1-netlab-f26-mechanism'
image = PREFIX + ':lab'
root = str(Path.cwd())
subprocess.run(['docker','run','-d','--name',PREFIX,'--privileged','--network','none',
                '-v',root+':/work:ro',image],check=True)
try:
    for mode in ('normal','padded','dump'):
        subprocess.run([sys.executable,'-B','docs/poc/G2-followup26-evidence/experiment.py',mode],check=True)
finally:
    subprocess.run(['docker','rm','-fv',PREFIX],check=True)
