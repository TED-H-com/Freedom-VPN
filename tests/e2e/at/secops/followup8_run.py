#!/usr/bin/env python3
"""Follow-up 8 foreground runner: AT30 live central restore & cross-system reconciliation."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import uuid

from run import (
    PORT_BASE,
    PREFIX,
    PRIVATE,
    PUBLIC,
    ROOT,
    command,
    inventory,
    summarize_go,
    utc,
    write_json,
)


def cleanup_extra_restore_objects(env: dict) -> None:
    """Remove any fvpn-w4-secops-restore-* or extra member containers, volumes, or images."""
    for c in [
        "fvpn-w4-secops-member-at30-m1-d1",
        "fvpn-w4-secops-member-at30-m2-d1",
        "fvpn-w4-secops-member-at30-m2-d2",
        "fvpn-w4-secops-member-at30-m2-d3-t1",
        "fvpn-w4-secops-member-at30-m3-d1-t1",
        "fvpn-w4-secops-restore-neg",
        "fvpn-w4-secops-restore-neg-init",
        "fvpn-w4-secops-restore-neg-live",
        "fvpn-w4-secops-restore-pitr",
        "fvpn-w4-secops-restore-pitr-init",
        "fvpn-w4-secops-restore-live",
    ]:
        subprocess.run(
            ["docker", "rm", "-fv", c],
            cwd=ROOT,
            env=env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
    for v in [
        "fvpn-w4-secops-member-at30-m1-d1",
        "fvpn-w4-secops-member-at30-m2-d1",
        "fvpn-w4-secops-member-at30-m2-d2",
        "fvpn-w4-secops-member-at30-m2-d3-t1",
        "fvpn-w4-secops-member-at30-m3-d1-t1",
        "fvpn-w4-secops-restore-neg-pg",
        "fvpn-w4-secops-restore-live-pg",
    ]:
        subprocess.run(
            ["docker", "volume", "rm", v],
            cwd=ROOT,
            env=env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
    subprocess.run(
        ["docker", "image", "rm", "fvpn-w4-secops-restore-image:17"],
        cwd=ROOT,
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def main() -> int:
    PRIVATE.mkdir(parents=True, exist_ok=True)
    (ROOT / ".agent/tmp").mkdir(parents=True, exist_ok=True)
    source = subprocess.check_output(
        ["git", "rev-parse", "HEAD"], cwd=ROOT, text=True
    ).strip()
    record = {
        "driver_run_id": str(uuid.uuid4()),
        "source_revision": source,
        "started_at": utc(),
        "expected": (
            "AT30 live central DB restore to T0 while node, payment provider, and NetBird hold T1 state; "
            "2-minute fail-safe negative before freeze/begin-recovery; 5-step fvpn-usage-admin restore protocol; "
            "retained-node usage replay (0 lost, 0 double-counted); provider and peer/revocation reconciliation; measured RPO/RTO"
        ),
        "prepare": {
            "reset": True,
            "prefix": PREFIX,
            "port_base": PORT_BASE,
        },
    }
    with tempfile.TemporaryDirectory(
        prefix="secops-f8.", dir=ROOT / ".agent/tmp"
    ) as scratch:
        env = dict(
            os.environ,
            PATH=str(Path.home() / ".local/go/bin") + ":" + os.environ.get("PATH", ""),
            GOTOOLCHAIN="local",
            CGO_ENABLED="0",
            PYTHONDONTWRITEBYTECODE="1",
            TMPDIR=scratch,
            GOTMPDIR=scratch,
            E2E_RESET="1",
            FVPN_E2E_PREFIX=PREFIX,
            FVPN_E2E_PORT_BASE=PORT_BASE,
            FVPN_E2E_STATE=str(ROOT / ".agent/e2e-state-w4-secops"),
        )
        before = inventory(env)
        record["inventory_before"] = before
        if any(v["count"] or v["exit_code"] for v in before.values()):
            raise SystemExit(
                "own namespace occupied; refusing to attach or tear down"
            )
        (PUBLIC / "followup8-live.json").unlink(missing_ok=True)
        write_json(PUBLIC / "followup8-run.json", record)
        try:
            print("follow-up 8: fresh e2e-up", flush=True)
            record["up"], _ = command(
                "followup8-e2e-up",
                ["make", "-f", "mk/e2e.mk", "e2e-up"],
                env,
                1500,
            )
            if record["up"]["exit_code"] != 0:
                cleanup_extra_restore_objects(env)
                command(
                    "followup8-e2e-down-retry",
                    ["make", "-f", "mk/e2e.mk", "e2e-down"],
                    env,
                    300,
                )
                __import__("time").sleep(3)
                record["up"], _ = command(
                    "followup8-e2e-up",
                    ["make", "-f", "mk/e2e.mk", "e2e-up"],
                    env,
                    1500,
                )
            if record["up"]["exit_code"] == 0:
                name = "TestAT30_Followup8_LiveRestoreReconcile"
                args = [
                    "go",
                    "test",
                    "-json",
                    "-tags",
                    "e2e,integration",
                    "-p",
                    "1",
                    "-parallel",
                    "2",
                    "-count=1",
                    "-timeout",
                    "35m",
                    "-run",
                    "^" + name + "$",
                    "./tests/e2e/at/secops",
                ]
                cmd, raw = command(
                    "followup8-live",
                    args,
                    dict(env, FVPN_SECOPS_FOLLOWUP8="1"),
                    2160,
                )
                record["live"] = dict(
                    cmd, **summarize_go(raw, [name], cmd["exit_code"])
                )
        finally:
            cleanup_extra_restore_objects(env)
            record["down"], _ = command(
                "followup8-e2e-down",
                ["make", "-f", "mk/e2e.mk", "e2e-down"],
                env,
                300,
            )
            cleanup_extra_restore_objects(env)
            record["inventory_after"] = inventory(env)
            record["finished_at"] = utc()
            write_json(PUBLIC / "followup8-run.json", record)

    subprocess.run(
        ["python3", "tests/e2e/at/secops/followup8_records.py"],
        cwd=ROOT,
        check=True,
    )
    verdict = json.loads((PUBLIC / "followup8-verdict.json").read_text())
    return 2 if verdict["blocked"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
