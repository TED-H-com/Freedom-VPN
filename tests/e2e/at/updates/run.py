#!/usr/bin/env python3
"""Run the owned AT extension in the shared devstack test package via Go overlay.

The virtual file never touches the parent package on disk. All release builds,
signing, registration, enrollment and service hooks use its existing helpers.
"""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[4]
DRIVER = ROOT / "tests/e2e/at/updates/live_test.go.txt"


def main():
    mode = sys.argv[1] if len(sys.argv) == 2 else ""
    if mode not in {"lint", "live", "compile"}:
        raise SystemExit("usage: run.py lint|live|compile")
    scratch = ROOT / ".agent/tmp"
    scratch.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="at-updates-overlay-", dir=scratch) as tmp:
        overlay = Path(tmp) / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {
            str(ROOT / "tests/e2e/at_updates_followup_test.go"): str(DRIVER)
        }}))
        env = dict(os.environ, TMPDIR=tmp, GOTMPDIR=tmp,
                   GOTOOLCHAIN="local", CGO_ENABLED="0")
        env["GOFLAGS"] = f"-overlay={overlay}"
        if mode == "lint":
            formatted = subprocess.check_output(["gofmt", "-l", str(DRIVER)], cwd=ROOT)
            if formatted:
                raise SystemExit("gofmt required: live_test.go.txt")
            subprocess.run(["go", "vet", "-tags", "e2e", "./tests/e2e"],
                           cwd=ROOT, env=env, check=True)
            subprocess.run(["go", "test", "-tags", "e2e", "-count=1",
                            "./tests/e2e", "-run", "^TestUpdatesFollowupEvidence$"],
                           cwd=ROOT, env=env, check=True)
            # Staticcheck's parser does not honor Go's virtual source files.
            # Analyze an ephemeral copy under our owned path (and thus within
            # tests/e2e/internal's import boundary), never modify shared files.
            with tempfile.TemporaryDirectory(prefix=".lint-", dir=DRIVER.parent) as copy:
                for source in (ROOT / "tests/e2e").glob("*.go"):
                    shutil.copyfile(source, Path(copy) / source.name)
                shutil.copyfile(DRIVER, Path(copy) / "at_updates_followup_test.go")
                env.pop("GOFLAGS")
                result = subprocess.run([
                    "go", "run", "honnef.co/go/tools/cmd/staticcheck@v0.8.2",
                    "-tags", "e2e", "./" + str(Path(copy).relative_to(ROOT))
                ], cwd=ROOT, env=env, check=False)
                return result.returncode
        else:
            if mode == "live":
                if env.get("FVPN_E2E_PREFIX") != "w4-updates" or env.get("FVPN_E2E_PORT_BASE") != "42000":
                    raise SystemExit("live run requires owned prefix w4-updates / 42000")
                env["FVPN_E2E"] = "1"
            commands = [["go", "test", "-tags", "e2e", "-count=1", "-v",
                         "-timeout", "45m", "-p", "1", "./tests/e2e", "-run",
                         "^TestUpdatesFollowup$" if mode == "live" else "^TestUpdatesFollowupEvidence$"]]
        for command in commands:
            result = subprocess.run(command, cwd=ROOT, env=env, check=False)
            if result.returncode:
                return result.returncode
    return 0


if __name__ == "__main__":
    sys.exit(main())
