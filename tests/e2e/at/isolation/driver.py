#!/usr/bin/env python3
"""Add the owned test to the shared driver package without copying its source."""
import json
import os
import shutil
from pathlib import Path
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parents[4]
os.chdir(root)
(root / ".agent/tmp").mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix="isolation-driver.", dir=root / ".agent/tmp") as temporary:
    overlay = Path(temporary) / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(root / "tests/e2e/isolation_owned_test.go"): str(root / "tests/e2e/at/isolation/driver.go.txt")}}))
    env = dict(os.environ, TMPDIR=temporary, GOTMPDIR=temporary, GOTOOLCHAIN="local", CGO_ENABLED="0")
    env["GOFLAGS"] = "-overlay=" + str(overlay)
    mode = sys.argv[1] if len(sys.argv) > 1 else "run"
    if mode == "lint":
        subprocess.run(["go", "vet", "-tags", "e2e", "./tests/e2e"], env=env, check=True)
        # Staticcheck's parser does not honor GOFLAGS overlays for virtual
        # filenames. Analyze an ephemeral copy under our owned package instead.
        with tempfile.TemporaryDirectory(prefix=".lint-", dir=root / "tests/e2e/at/isolation") as lintdir:
            for source in (root / "tests/e2e").glob("*.go"):
                shutil.copyfile(source, Path(lintdir) / source.name)
            shutil.copyfile(root / "tests/e2e/at/isolation/driver.go.txt", Path(lintdir) / "isolation_owned_test.go")
            lintenv = dict(env, GOFLAGS="")
            result = subprocess.run(["go", "run", "honnef.co/go/tools/cmd/staticcheck@v0.8.2", "-tags", "e2e", "./" + str(Path(lintdir).relative_to(root))], env=lintenv)
        raise SystemExit(result.returncode)
    elif mode == "build":
        commands = [["go", "test", "-c", "-tags", "e2e", "-o", temporary+"/driver.test", "./tests/e2e"]]
    else:
        env.update(FVPN_E2E="1", FVPN_E2E_PREFIX="w4-isolation", FVPN_E2E_PORT_BASE="36000", FVPN_E2E_STATE=str(root/".agent/e2e-state-w4-isolation"))
        commands = [["go", "test", "-tags", "e2e", "-count=1", "-timeout=55m", "-run=^TestIsolationAssembled$", "-v", "./tests/e2e"]]
    for command in commands:
        result = subprocess.run(command, env=env)
        if result.returncode:
            raise SystemExit(result.returncode)
