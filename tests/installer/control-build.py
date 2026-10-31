#!/usr/bin/env python3
"""Build/lint the TEST ONLY control using the shared node-internal fixture.

A temporary harness module has a node/ import path, allowing Go's internal
import. It uses the real root module through a local replace, with the same
pinned dependencies. Only installer-owned fixture sources are copied. No node source or
directory is written; all generated files live on disk in .agent/tmp.
"""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile


def main():
    root = Path(__file__).resolve().parents[2]
    scratch_root = root / ".agent/tmp"
    scratch_root.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="installer-control-", dir=scratch_root) as scratch:
        temp = Path(scratch) / "source"
        temp.mkdir()
        module = "github.com/TED-H-com/Freedom-VPN"
        manifest = (root / "go.mod").read_text().replace(
            "module " + module, "module " + module + "/node/installerfixture", 1)
        manifest += f"\nrequire {module} v0.0.0\nreplace {module} => {json.dumps(str(root))}\n"
        (temp / "go.mod").write_text(manifest)
        (temp / "go.sum").write_bytes((root / "go.sum").read_bytes())
        for source in (root / "tests/installer/cmd/fvpn-w5-control").glob("*.go"):
            (temp / source.name).write_bytes(source.read_bytes())
        env = dict(os.environ, TMPDIR=scratch, GOTMPDIR=scratch, GOWORK="off")
        package = "."
        go = env.get("INSTALLER_GO", "go")
        if sys.argv[1] == "build":
            commands = [[go, "build", "-tags", "installerfixture", "-trimpath",
                         "-o", sys.argv[2], package]]
        elif sys.argv[1] == "lint":
            commands = [
                [go, "vet", "-tags", "installerfixture", package],
                [go, "run", "honnef.co/go/tools/cmd/staticcheck@v0.8.2",
                 "-tags", "installerfixture", package],
            ]
        else:
            raise SystemExit("expected build <output> or lint")
        for command in commands:
            result = subprocess.run(command, cwd=temp, env=env, check=False)
            if result.returncode:
                return result.returncode
    return 0


if __name__ == "__main__":
    sys.exit(main())
