#!/usr/bin/env python3
"""Run the fixed drill command and require its test2json execution inventory."""

import json
import subprocess
import sys


PACKAGE = "github.com/TED-H-com/Freedom-VPN/tests/backup"
# Independent of the make -run expression: deleting a selection must fail here.
REQUIRED = (
    "TestAT30_BackupPITRDrill",
    "TestAT30_IsolatedHostConfig",
    "TestAT30_IsolatedHostConfig/time",
    "TestAT30_IsolatedHostConfig/lsn",
    "TestAT30_IsolatedHostConfig/latest",
    "TestAT30_IsolatedHostConfig/direct_failure_cleanup",
    "TestAT30_IsolatedHostConfig/preexisting_volume",
    "TestAT30_BackupLocaleGuards",
    "TestAT30_BackupLocaleGuards/C/all",
    "TestAT30_BackupLocaleGuards/C.UTF-8/all",
    "TestAT30_BackupLocaleGuards/en_US.UTF-8/all",
    "TestAT30_BackupLocaleGuards/en_US.UTF-8/lang",
    "TestAT30_BackupLocaleGuards/en_US.UTF-8/collate",
    "TestAT30_BackupLocaleGuards/en_US.UTF-8/ctype",
)


def main(command):
    actions = {name: [] for name in REQUIRED}
    package_passed = False
    bad_event = False
    failed = False
    with subprocess.Popen(command, stdout=subprocess.PIPE, text=True) as process:
        for line in process.stdout:
            try:
                event = json.loads(line)
                if not isinstance(event, dict):
                    raise ValueError("event is not an object")
            except ValueError:
                print("backup-drill: invalid test event: " + line.rstrip(), flush=True)
                bad_event = True
                continue
            if "Output" in event:
                print(event["Output"], end="", flush=True)
            if event.get("Action") in ("fail", "build-fail"):
                failed = True
                if event["Action"] == "build-fail":
                    print("backup-drill: build-fail event", flush=True)
            if event.get("Package") != PACKAGE:
                continue
            name = event.get("Test")
            action = event.get("Action")
            if name in actions and action in ("run", "pass", "fail", "skip"):
                actions[name].append(action)
            if name is None and action == "pass":
                package_passed = True
        status = process.wait()
    missing = [name for name, seen in actions.items() if seen != ["run", "pass"]]
    for name in missing:
        print(f"backup-drill: required test did not run and pass: {name} (events={actions[name]})")
    if status != 0:
        print(f"backup-drill: go test exited {status}")
    if not package_passed:
        print(f"backup-drill: missing package pass: {PACKAGE}")
    if missing or bad_event or failed or status != 0 or not package_passed:
        return 1
    print(f"backup-drill: required inventory PASS ({len(REQUIRED)} tests/subtests)")
    return 0


if __name__ == "__main__":
    if len(sys.argv) < 3 or sys.argv[1] != "--":
        sys.exit("usage: check_drill.py -- go test -json ...")
    sys.exit(main(sys.argv[2:]))
