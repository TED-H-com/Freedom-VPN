#!/usr/bin/env python3
"""Use the shared lifecycle, keeping its mutable cache tags task-owned."""
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parents[4]
os.chdir(root)
if sys.argv[1:] not in [["up"], ["down"]]:
    raise SystemExit("usage: stack.py up|down")
env = dict(os.environ, FVPN_E2E_PREFIX="w4-isolation", FVPN_E2E_PORT_BASE="36000",
           FVPN_E2E_STATE=str(root / ".agent/e2e-state-w4-isolation"), E2E_RESET="1")
for service, tag in [("nginx", "1.30.0"), ("coredns", "1.12.0"), ("adguard", "v0.107.59"), ("netbird", "0.82.0")]:
    env["FVPN_E2E_IMAGE_" + service.upper()] = "fvpn-w4-isolation-cache-" + service + ":" + tag
(root / ".agent/tmp").mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix="isolation-docker.", dir=root / ".agent/tmp") as temporary:
    docker = shutil.which("docker")
    shim = Path(temporary) / "docker"
    shim.write_text("#!/usr/bin/env python3\nimport os,sys\nargs=[a.replace('fvpn-devstack-', 'fvpn-w4-isolation-cache-') if a.startswith('fvpn-devstack-') else a for a in sys.argv[1:]]\nos.execv(" + repr(docker) + ", ['docker']+args)\n")
    shim.chmod(0o755)
    env["PATH"] = temporary + os.pathsep + env["PATH"]
    result = subprocess.run(["sh", "deploy/devstack/" + sys.argv[1] + ".sh"], env=env)
raise SystemExit(result.returncode)
