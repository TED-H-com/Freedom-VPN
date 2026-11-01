#!/usr/bin/env python3
"""TEST ONLY: summarize retained F24 measurements without running the lab."""
import hashlib
import gzip
import json
import pathlib
import runpy

HERE = pathlib.Path(__file__).resolve().parent
OUT = HERE.parent / "comparison"
api = runpy.run_path(str(HERE / "compare.py"))
write, stats = api["write"], api["stats"]


def read(path):
    return json.loads(path.read_text())


def raw_bytes(path):
    archive = path.with_name(path.name + ".raw.gz")
    return gzip.decompress(archive.read_bytes()) if archive.exists() else path.read_bytes()


def host_summary(path):
    row = read(path)
    return dict(
        receipt=path.name, quiet=row["quiet"],
        max_r=max(r["r"] for r in row["intervals"]),
        min_idle=min(r["id"] for r in row["intervals"]),
        load_before=row["load_before"], load_after=row["load_after"],
        d_state_before=row["d_state_before"], d_state_after=row["d_state_after"],
        process_count_before=len(row["processes_before"]),
        process_count_after=len(row["processes_after"]),
    )


def reload_rows(attempt):
    path = OUT / attempt / "reload/TestAT10_AT13_ReloadRevokeBaseline/at10-apply.jsonl"
    return [dict(index=i, exit=0 if row["timing"]["succeeded"] else 1, result=row)
            for i, row in enumerate(json.loads(s) for s in path.read_text().splitlines())]


def main():
    # The executing harness used .json filenames for tc output which turned
    # out to be text. Retain exactly those bytes under the correct suffix.
    receipt = OUT / "tc-text-receipt.json"
    renamed = read(receipt) if receipt.exists() else dict(
        test_only=True, note="Only filename suffixes changed; tc text bytes are retained exactly.", files=[])
    for path in sorted(OUT.glob("*/tc-*.json")):
        data = path.read_bytes()
        target = path.with_suffix(".txt")
        if target.exists():
            raise RuntimeError("refusing to overwrite " + str(target))
        path.rename(target)
        renamed["files"].append(dict(original=str(path.relative_to(OUT)),
                                     retained=str(target.relative_to(OUT)),
                                     sha256=hashlib.sha256(data).hexdigest(), bytes=len(data)))
    write(receipt, renamed)

    completed = {}
    attempts = []
    for leg in ["A1", "B1", "A2", "B2"]:
        paths = sorted(OUT.glob(leg + "-attempt*/status.json"),
                       key=lambda p: int(p.parent.name.split("attempt")[1]))
        for path in paths:
            status = read(path)
            key = path.parent.name
            attempts.append(dict(attempt=key, **status,
                                 postflight=host_summary(OUT / "host" / (key + "-postflight.json"))))
            if status["postflight_quiet"]:
                if leg in completed:
                    raise RuntimeError("multiple qualified attempts for " + leg)
                completed[leg] = key
    api["summarize"](completed)
    pooled_reload = {arm: [] for arm in ["A", "B"]}
    qualified_host = {}
    ordinary_command_counts = []
    tc_hashes = set()
    for leg, attempt in completed.items():
        pooled_reload[leg[0]].extend(reload_rows(attempt))
        ordinary_command_counts.extend(len(r["result"]["timing"].get("commands") or [])
                                       for r in read(OUT / attempt / "applies.json")["records"])
        tc_hashes.update(hashlib.sha256(raw_bytes(OUT / attempt / name)).hexdigest()
                         for name in ["tc-fv-ih.txt", "tc-fv-fh.txt"])
        qualified_host[leg] = [host_summary(p) for p in sorted((OUT / "host").glob(attempt + "-*.json"),
                                                             key=lambda p: int(read(p)["start_ns"]))
                               if read(p)["quiet"]]
    complete = len(completed) == 4
    failures = [r["attempt"] for r in attempts
                if r["postflight_quiet"] and (r["reload_exit"] or r["failures"])]
    host_rows = [read(p) for p in (OUT / "host").glob("*.json")]
    window = read(OUT / "window.json")
    write(OUT / "assessment.json", dict(
        test_only=True, completed=completed, complete_quartet=complete,
        host_checks=len(host_rows),
        measured_window_seconds=(max(int(r["end_ns"]) for r in host_rows)
                                 - int(window["start_ns"])) / 1e9,
        ordinary_samples_without_lifecycle_commands=sum(n == 0 for n in ordinary_command_counts),
        distinct_tc_shapes=sorted(tc_hashes),
        qualified_functional_failures=failures, attempts=attempts,
        qualified_host_checks=qualified_host,
        pooled_reload={arm: stats(rows) for arm, rows in pooled_reload.items()},
        result=("complete; no qualified functional failure" if complete and not failures
                else "complete; qualified functional failure" if complete and failures
                else "blocked: host not quiet"),
        interpretation="D-138: pool every qualified leg per arm. This is one window; no causal timing shift is attributed. Excluded attempts remain excluded, and historical failures remain failed.",
    ))
    write(OUT / "analysis-provenance.json", dict(
        test_only=True,
        note="Measurement used compare-run.py; summary field names and tc suffix corrected after execution without changing measured bytes.",
        initial_summary_error="KeyError: script after all four qualified legs; measurement cleanup completed, corrected summary regenerated from retained inputs.",
        files={name: hashlib.sha256((HERE / name).read_bytes()).hexdigest()
               for name in ["compare-initial.py", "compare-run.py", "compare.py", "analyze.py"]},
    ))


if __name__ == "__main__":
    main()
