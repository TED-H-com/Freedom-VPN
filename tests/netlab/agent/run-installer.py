"""Run the unchanged installer proof using only this lane's Docker names.

The adapter translates the installer's hard-coded object prefix on arguments
and output. It changes no commands, fixtures, files, assertions or health data.
"""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    docker = shutil.which("docker")
    if not docker:
        raise SystemExit("docker is required")
    scratch = Path.cwd() / ".agent" / "tmp"
    scratch.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=scratch, prefix="installer-") as temporary:
        shim = Path(temporary) / "docker"
        shim.write_text("#!/usr/bin/env python3\n"
                        "import subprocess,sys\n"
                        "old='fvpn-w5-node-installer'\n"
                        "new='fvpn-w2-agent-core-lab-installer'\n"
                        f"p=subprocess.run([{docker!r}]+[x.replace(old,new) for x in sys.argv[1:]], stdout=subprocess.PIPE)\n"
                        "sys.stdout.buffer.write(p.stdout.replace(new.encode(),old.encode()))\n"
                        "sys.exit(p.returncode)\n")
        shim.chmod(0o700)
        env = dict(os.environ, PATH=temporary + os.pathsep + os.environ["PATH"],
                   TMPDIR=temporary, GOTMPDIR=temporary, CGO_ENABLED="0", GOTOOLCHAIN="local")
        result = subprocess.run(["go", "test", "-tags", "installer", "-count=1", "-v",
                                 "-timeout=15m", "-run", "^TestD089_RealSeedThenJournaledUpgrade$",
                                 "./tests/installer"], env=env)
        raise SystemExit(result.returncode)


if __name__ == "__main__":
    main()
