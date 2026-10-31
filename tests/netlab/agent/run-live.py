"""Foreground, disk-backed live agent lab. No host network changes."""
import os
from pathlib import Path
import subprocess
import tempfile


def main():
    scratch = Path.cwd() / ".agent" / "tmp"
    scratch.mkdir(parents=True, exist_ok=True)
    name = f"fvpn-w2-agent-core-lab-live-{os.getpid()}"
    image = name + ":lab"
    evidence = Path.cwd() / ".agent" / "f25-live-evidence"
    evidence.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=scratch, prefix="live-") as temporary:
        env = dict(os.environ, GOMAXPROCS="4", GOFLAGS="-p=4", CGO_ENABLED="0",
                   GOTOOLCHAIN="local", TMPDIR=temporary, GOTMPDIR=temporary)
        binary = str(Path(temporary) / "agent.test")
        def run(*args, **kwargs):
            subprocess.run(args, check=True, **kwargs)
        try:
            run("go", "test", "-c", "-tags", "netlab", "-o", binary, "./node/agent", env=env)
            run("docker", "build", "-t", image, "tests/netlab")
            # Scratch is in the container's disk overlay, removed with it.
            run("docker", "run", "--rm", "--name", name, "--privileged", "--network", "none", "--workdir", "/scratch",
                "-e", "FVPN_AGENT_LIVE_NETLAB=1", "-e", "FVPN_AGENT_EVIDENCE_DIR=/evidence",
                "-v", str(evidence) + ":/evidence", "-e", "GOMAXPROCS=4", "-e", "TMPDIR=/scratch",
                "-v", binary + ":/agent.test:ro", image, "/agent.test", "-test.run",
                "^TestD085_LiveReadNetlab$", "-test.v", "-test.timeout=3m")
        finally:
            subprocess.run(["docker", "rm", "-fv", name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            subprocess.run(["docker", "rmi", image], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


if __name__ == "__main__":
    main()
