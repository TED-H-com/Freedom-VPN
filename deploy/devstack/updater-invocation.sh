#!/bin/sh
# TEST ONLY: retain every invocation before publishing the latest compatibility
# files. This recorder neither takes update.lock nor changes an updater gate.
set -eu
root=$1
shift
mkdir -p "$root/updater-invocations"
receipt=$(mktemp -d "$root/updater-invocations/invocation.XXXXXXXX")
started=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
printf '{"started_at":"%s"}\n' "$started" > "$receipt/start.json"
set +e
sh -c 'printf "%s\n" "$$" > "$1/pid"; shift; exec "$@"' invocation "$receipt" "$@" > "$receipt/output.txt" 2>&1
code=$?
set -e
ended=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
printf '{"invocation_id":"%s","started_at":"%s","ended_at":"%s","pid":%s,"exit_code":%s}\n' \
    "${receipt##*/}" "$started" "$ended" "$(cat "$receipt/pid")" "$code" > "$receipt/result.json"
cp "$receipt/output.txt" "$root/updater.out"
cp "$receipt/result.json" "$root/updater-result.json.tmp"
mv "$root/updater-result.json.tmp" "$root/updater-result.json"
