#!/usr/bin/env python3
"""Copy repo 1 using the host pgBackRest configuration; keep ciphertext and mtimes.

The caller supplies a new disk directory. A concurrent expiry or changed object
fails the copy/verification; retry with a fresh directory. No partial copy passes.
"""
import json
import os
from pathlib import Path, PurePosixPath
import subprocess
import sys

stanza, destination = sys.argv[1:]
root = Path(destination)
root.mkdir(mode=0o755)  # exclusive: never replace an existing repository
command = ["pgbackrest", f"--stanza={stanza}", "--repo=1"]
for subtree in (f"archive/{stanza}", f"backup/{stanza}"):
    listing = json.loads(subprocess.check_output(
        command + ["--recurse", "--output=json", "repo-ls", subtree]))
    for name, item in listing.items():
        if item["type"] != "file":
            continue  # pgBackRest's 'latest' symlink is not needed for restore
        relative = PurePosixPath(subtree) / name
        if relative.is_absolute() or ".." in relative.parts:
            raise ValueError("unsafe repository path")
        path = root / relative
        path.parent.mkdir(parents=True, exist_ok=True, mode=0o755)
        with path.open("xb") as output:
            subprocess.run(command + ["--raw", "repo-get", str(relative)],
                           stdout=output, check=True)
        if path.stat().st_size != item["size"]:
            raise ValueError("repository changed during copy; retry")
        os.chmod(path, 0o644)
        os.utime(path, (item["time"], item["time"]))
