"""Foreground, disk-backed live agent lab. No host network changes."""
import os
from pathlib import Path
import subprocess
import tempfile


def main():
    scratch = Path.cwd() / ".agent" / "tmp"
    scratch.mkdir(parents=True, exist_ok=True)
    prefix = os.environ.get("FVPN_LAB_PREFIX", "fvpn-w2-agent-core-lab")
    if not prefix.startswith("fvpn-w2-agent-core-"):
        raise ValueError("lane-owned Docker prefix required")
    name = f"{prefix}-live-{os.getpid()}"
    image = name + ":lab"
    evidence = Path(os.environ.get("FVPN_AGENT_EVIDENCE_DIR", str(Path.cwd() / ".agent" / "f27-live-evidence"))).resolve()
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
                "-e", "FVPN_F29_SIBLING",
                "-v", str(evidence) + ":/evidence", "-e", "GOMAXPROCS=4", "-e", "TMPDIR=/scratch",
                "-v", binary + ":/agent.test:ro", image, "/agent.test", "-test.run",
                "^TestD085_LiveReadNetlab$", "-test.v", "-test.timeout=3m")
        finally:
            # The privileged test writes private (0600) diagnostic files as
            # container root. Hand them back to this harness before removing
            # its image, including on a failed test. Only this run's bind mount.
            try:
                subprocess.run(["docker", "run", "--rm", "--name", name + "-artifacts",
                                "--network", "none", "-v", str(evidence) + ":/evidence",
                                image, "chown", "-R", f"{os.getuid()}:{os.getgid()}", "/evidence"],
                               check=True)
            finally:
                subprocess.run(["docker", "rm", "-fv", name, name + "-artifacts"],
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                subprocess.run(["docker", "rmi", image], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


if __name__ == "__main__":
    main()
