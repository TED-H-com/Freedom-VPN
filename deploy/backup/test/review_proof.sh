#!/usr/bin/env bash
# Adversarial proofs for the PR #86 backup / PITR deliverable.
# It prints FAIL for each defect and PASS for each check that held.
# Containers, volumes and images it creates are named fvpn-w4-ops-backups-review-<random>-*.
# It never starts, stops, removes or execs any other container.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
PREFIX="fvpn-w4-ops-backups-review-$(od -An -N4 -tx1 /dev/urandom | tr -d ' \n')"
IMAGE="${PREFIX}-postgres:17"
# TEST ONLY - NOT A SECRET
CIPHER="review-cipher-not-a-secret"
mkdir -p "${ROOT}/.agent/tmp"
LOG_DIR="$(mktemp -d "${ROOT}/.agent/tmp/${PREFIX}.XXXXXX")"
export TMPDIR="$LOG_DIR" GOTMPDIR="$LOG_DIR"
mkdir -p "$LOG_DIR"
FAILS=0

fail() {
	FAILS=$((FAILS + 1))
	printf 'FAIL %s\n' "$*"
}
pass() {
	printf 'PASS %s\n' "$*"
}
section() {
	printf '\n==== %s ====\n' "$*"
}

LIVE_TMP=""
WRAP_BIN=""
# Invoked through the EXIT trap; ShellCheck cannot follow this indirect call.
# shellcheck disable=SC2317
cleanup() {
	local name
	while read -r name; do
		[[ -n "$name" ]] || continue
		docker rm -f "$name" >/dev/null 2>&1 || true
	done < <(docker ps -a --format '{{.Names}}' | grep "^${PREFIX}-" || true)
	while read -r name; do
		[[ -n "$name" ]] || continue
		docker volume rm -f "$name" >/dev/null 2>&1 || true
	done < <(docker volume ls --format '{{.Name}}' | grep "^${PREFIX}-" || true)
	docker rmi "$IMAGE" >/dev/null 2>&1 || true
	rm -rf "$LOG_DIR"
}
trap cleanup EXIT

docker_real="$(command -v docker)"
WRAP_BIN="$(mktemp -d "${LOG_DIR}/${PREFIX}-docker-wrap.XXXXXX")"
cat >"${WRAP_BIN}/docker" <<EOF
#!/bin/bash
printf '%s\n' "\$*" >> "${LOG_DIR}/docker-argv.log"
exec ${docker_real} "\$@"
EOF
chmod 0755 "${WRAP_BIN}/docker"
export PATH="${WRAP_BIN}:${PATH}"

wait_primary() {
	local container="$1" logs
	for _ in $(seq 1 240); do
		logs="$(docker logs "$container" 2>&1)"
		if [[ "$logs" == *"PostgreSQL init process complete; ready for start up."* ]] &&
			[[ "${logs##*PostgreSQL init process complete; ready for start up.}" == *"database system is ready to accept connections"* ]] &&
			docker exec --user postgres "$container" psql -U fvpn -d fvpn -tAc 'SELECT 1' >/dev/null 2>&1; then
			sleep 1
			docker exec --user postgres "$container" psql -U fvpn -d fvpn -tAc 'SELECT 1' >/dev/null
			return
		fi
		sleep 0.5
	done
	fail "final PostgreSQL server did not become ready: $container"
	return 1
}

section "static: secrets are not echoed, but the cipher is placed on argv"
if grep -RInE 'echo .*(PGBACKREST_REPO1_S3_KEY|PGBACKREST_REPO1_CIPHER_PASS|CIPHER_PASS)' \
	"${ROOT}/deploy/backup" "${ROOT}/.github/workflows/backup.yml" \
	--exclude-dir=test >"${LOG_DIR}/${PREFIX}-echo-grep.txt"; then
	fail "script echoes a secret env var: $(cat "${LOG_DIR}/${PREFIX}-echo-grep.txt")"
else
	pass "no echo of cipher/S3 env vars in deploy/backup scripts or backup.yml"
fi
if grep -n 'repo1-cipher-pass=' "${ROOT}/deploy/backup/restore-pitr.sh" | grep -q 'CIPHER_PASS'; then
	fail "restore-pitr.sh puts the cipher passphrase on the pgbackrest/docker argument list (restore-pitr.sh:251)"
else
	pass "cipher passphrase stays off the argument list"
fi
if grep -n 'archive-async=y' "${ROOT}/deploy/backup/pgbackrest.conf.example" >/dev/null; then
	fail "production pgbackrest.conf.example sets archive-async=y (WAL spool success is reported to PostgreSQL before the off-host push)"
else
	pass "production profile does not enable archive-async"
fi
if grep -RInE 'pgbackrest[[:space:]]+verify|pgbackrest[[:space:]].*[[:space:]]verify([[:space:]]|$)' \
	"${ROOT}/deploy/backup" "${ROOT}/.github/workflows/backup.yml" "${ROOT}/mk/backup.mk" \
	--exclude-dir=test >/dev/null; then
	pass "a pgbackrest verify step is wired into the backup path"
else
	fail "backup.sh, verify.sh, backup.yml and backup.mk never run 'pgbackrest verify'"
fi
if grep -n 'continue-on-error' "${ROOT}/.github/workflows/backup.yml" >/dev/null; then
	fail "backup.yml sets continue-on-error"
else
	pass "backup.yml has no continue-on-error"
fi
if grep -n 'rpoArchiveLagMs := time.Since(walSwitchStart)' "${ROOT}/tests/backup/drill_integration_test.go" >/dev/null; then
	fail "TestAT30_BackupPITRDrill records measured_rpo_archive_lag_ms as time.Since(wal-switch), not pg_stat_archiver lag or archive_timeout"
else
	pass "drill RPO is not a wal-switch stopwatch"
fi
if grep -n 'COALESCE(last_archived_time, clock_timestamp())' "${ROOT}/deploy/backup/verify.sh" >/dev/null \
	&& grep -n 'RPO_MS=0' "${ROOT}/deploy/backup/verify.sh" >/dev/null; then
	fail "verify.sh reports rpo_ms=0 when no primary is given, and treats a NULL last_archived_time as zero lag"
else
	pass "verify.sh archive-lag measurement does not collapse to zero"
fi
if grep -n 'repo1-retention-diff=14' "${ROOT}/deploy/backup/pgbackrest.conf.example" >/dev/null \
	&& grep -n '14 days of differential' "${ROOT}/docs/ops/backups.md" >/dev/null; then
	fail "docs/ops/backups.md calls repo1-retention-diff=14 a 14-day differential window; pgBackRest interprets it as a count of 14 diffs"
else
	pass "differential retention units match the docs"
fi
if grep -n 'repo1-retention-archive=35' "${ROOT}/deploy/backup/pgbackrest.conf.example" >/dev/null \
	&& grep -n 'repo1-retention-archive-type=full' "${ROOT}/deploy/backup/pgbackrest.conf.example" >/dev/null \
	&& grep -n 'OnCalendar=Sun' "${ROOT}/deploy/backup/systemd/fvpn-backup-full.timer" >/dev/null; then
	fail "production archive retention keeps WAL for 35 full backups while full backups are weekly and expired by time at 35 days, so the 35-full WAL rule is not a 35-day expiry"
else
	pass "archive retention matches the documented 35-day window"
fi
if ! grep -nE -- '--primary-container|--primary-host' "${ROOT}/deploy/backup/systemd/fvpn-backup-verify.service" >/dev/null; then
	fail "fvpn-backup-verify.service does not pass --primary-container, so its published rpo_ms is the zero default"
else
	pass "verify unit measures a primary"
fi
if grep -n 'pgbackrest --stanza=fvpn --type=time' "${ROOT}/docs/handover/restore-drill.md" >/dev/null \
	&& ! grep -n 'restore-pitr.sh' "${ROOT}/docs/handover/restore-drill.md" >/dev/null; then
	fail "docs/handover/restore-drill.md Phase 2.1 runs raw pgbackrest restore with no --pg1-path and does not call restore-pitr.sh"
else
	pass "handover PITR command is the guarded restore script"
fi

section "static: handover SQL names objects the migrations do not create"
# Original review lines 132-153 searched for obsolete names in migrations,
# irrespective of whether the handover still used them. Execute the actual
# handover SQL below; keep this static control against the obsolete SQL names.
if grep -E 'billing_subscriptions|dns_resolver_states|target_resource|SELECT .*missing_ranges' "${ROOT}/docs/handover/restore-drill.md"; then
	fail "handover still contains obsolete SQL objects"
else
	pass "handover SQL uses current schema objects"
fi

section "guard: same directory and postmaster.pid are refused; a child directory is not"
LIVE_TMP="$(mktemp -d "${LOG_DIR}/${PREFIX}-live.XXXXXX")"
mkdir -p "${LIVE_TMP}/base" "${LIVE_TMP}/pg_wal"
printf '1234\n' >"${LIVE_TMP}/postmaster.pid"
printf '17\n' >"${LIVE_TMP}/PG_VERSION"
printf 'keep\n' >"${LIVE_TMP}/KEEP"
same_out="$LOG_DIR/same-dir.txt"
set +e
bash "${ROOT}/deploy/backup/restore-pitr.sh" --latest --target-dir "$LIVE_TMP" --primary-dir "$LIVE_TMP" >"$same_out" 2>&1
same_rc=$?
set -e
if [[ "$same_rc" -ne 0 ]] && grep -q 'refusing' "$same_out" && [[ -f "${LIVE_TMP}/KEEP" ]]; then
	pass "restore-pitr.sh refuses --target-dir equal to --primary-dir (rc=${same_rc})"
else
	fail "same-directory restore was not refused: rc=${same_rc} $(cat "$same_out")"
fi
child="${LIVE_TMP}/nested-restore"
child_out="$LOG_DIR/child-dir.txt"
set +e
bash "${ROOT}/deploy/backup/restore-pitr.sh" --latest --target-dir "$child" --primary-dir "$LIVE_TMP" >"$child_out" 2>&1
child_rc=$?
set -e
if [[ -d "$child" ]]; then
	fail "restore-pitr.sh created ${child} inside the live data directory before invoking pgbackrest (rc=${child_rc})"
	printf 'child stderr:\n%s\n' "$(cat "$child_out")"
else
	pass "child of the live data directory was refused (rc=${child_rc})"
fi

section "build review postgres image"
docker build -t "$IMAGE" "${ROOT}/deploy/backup/postgres" >"$LOG_DIR/docker-build.txt" 2>&1

section "RPO SQL: NULL last_archived_time is reported as zero lag"
docker run -d --name "${PREFIX}-nullrpo" --network none \
	-e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_USER=fvpn -e POSTGRES_DB=fvpn \
	-e FVPN_PGBACKREST_AUTO_STANZA=0 "$IMAGE" postgres -c archive_mode=off >"$LOG_DIR/nullrpo-id.txt"
wait_primary "${PREFIX}-nullrpo"
rpo_sql="SELECT COALESCE(GREATEST(0, (EXTRACT(EPOCH FROM (clock_timestamp() - COALESCE(last_archived_time, clock_timestamp()))) * 1000)::bigint), 0) FROM pg_stat_archiver;"
null_rpo="$(docker exec --user postgres "${PREFIX}-nullrpo" psql -U fvpn -d fvpn -tAc "$rpo_sql" | tr -d '[:space:]')"
null_raw="$(docker exec --user postgres "${PREFIX}-nullrpo" psql -U fvpn -d fvpn -tAc "SELECT archive_mode, last_archived_time IS NULL, archived_count FROM pg_stat_archiver, (SELECT current_setting('archive_mode') AS archive_mode) s;" | tr -d '[:space:]')"
printf 'null-archive query -> rpo_ms=%s detail=%s\n' "$null_rpo" "$null_raw"
# Original lines 203-212 embedded the broken SQL, so changing verify.sh could
# never make it pass. The old expression must reproduce the false zero.
if [[ "$null_rpo" == "0" && "$null_raw" == "off|t|0" ]]; then
	pass "negative control: old SQL falsely reports zero on a never-archived cluster"
else
	fail "never-archived negative control did not reproduce"
fi
null_fixed="$(docker exec -i --user postgres "${PREFIX}-nullrpo" psql -U fvpn -d fvpn -tA <"${ROOT}/deploy/backup/archive-state.sql")"
if [[ -z "$null_fixed" ]]; then
	pass "shipped archive SQL preserves unknown state"
else
	fail "shipped archive SQL invented a lag for NULL last_archived_time"
fi
if bash "${ROOT}/deploy/backup/verify.sh" --primary-container "${PREFIX}-nullrpo" \
	--repo-volume "${PREFIX}-empty-repo" --image "$IMAGE" --verify-prefix "${PREFIX}-null-verify" \
	--migrate-bin /unused >"$LOG_DIR/null-verify.txt" 2>&1; then
	fail "verify accepted a never-archived primary"
elif grep -q 'status=PASS' "$LOG_DIR/null-verify.txt"; then
	fail "verify emitted PASS for unknown archive state"
else
	pass "verify rejects a never-archived primary without PASS"
fi
docker rm -f "${PREFIX}-nullrpo" >/dev/null

section "volume guard: stopped container is held; removed container's volume is wiped"
docker volume create "${PREFIX}-held" >/dev/null
docker volume create "${PREFIX}-orphan" >/dev/null
docker volume create "${PREFIX}-empty-repo" >/dev/null
docker run -d --name "${PREFIX}-heldsrc" --network none \
	-v "${PREFIX}-held:/var/lib/postgresql/data" \
	-e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_USER=fvpn -e POSTGRES_DB=fvpn \
	"$IMAGE" >/dev/null
wait_primary "${PREFIX}-heldsrc"
docker exec --user postgres "${PREFIX}-heldsrc" bash -c 'echo held-live > /var/lib/postgresql/data/LIVE_MARKER'
docker stop "${PREFIX}-heldsrc" >/dev/null
held_out="$LOG_DIR/held-volume.txt"
set +e
FVPN_PGBACKREST_CIPHER_PASS="$CIPHER" bash "${ROOT}/deploy/backup/restore-pitr.sh" \
	--latest \
	--isolated-container "${PREFIX}-held-restore" \
	--target-volume "${PREFIX}-held" \
	--repo-volume "${PREFIX}-empty-repo" \
	--image "$IMAGE" >"$held_out" 2>&1
held_rc=$?
set -e
if [[ "$held_rc" -ne 0 ]] && grep -q 'mounted by container' "$held_out"; then
	pass "stopped container still protects its volume (rc=${held_rc})"
else
	fail "stopped container volume was not protected: rc=${held_rc} $(tail -n 20 "$held_out")"
fi
docker rm -f "${PREFIX}-heldsrc" >/dev/null
# Volume is now unattached but still holds the cluster. This is the fvpn-pg shape
# after the container object is removed and the anonymous volume is left behind.
orphan_out="$LOG_DIR/orphan-volume.txt"
set +e
FVPN_PGBACKREST_CIPHER_PASS="$CIPHER" bash "${ROOT}/deploy/backup/restore-pitr.sh" \
	--latest \
	--isolated-container "${PREFIX}-orphan-restore" \
	--target-volume "${PREFIX}-held" \
	--repo-volume "${PREFIX}-empty-repo" \
	--image "$IMAGE" >"$orphan_out" 2>&1
orphan_rc=$?
set -e
marker_after="$(docker run --rm --name "${PREFIX}-marker-read" --network none --entrypoint cat \
	-v "${PREFIX}-held:/var/lib/postgresql/data:ro" "$IMAGE" /var/lib/postgresql/data/LIVE_MARKER 2>/dev/null || true)"
printf 'orphan restore rc=%s marker_after=%q\n' "$orphan_rc" "$marker_after"
if [[ "$orphan_rc" -eq 0 || "$marker_after" != held-live ]]; then
	fail "restore-pitr.sh deleted LIVE_MARKER from an unmounted volume that still held a database (rc=${orphan_rc})"
else
	pass "unmounted volume contents survived (marker=${marker_after})"
fi
if grep -F -- "--repo1-cipher-pass=${CIPHER}" "$LOG_DIR/docker-argv.log" >/dev/null; then
	fail "docker argv log contains --repo1-cipher-pass=${CIPHER}"
else
	pass "cipher passphrase did not appear on docker argv"
fi

section "PITR: base-backup row, post-backup pre-target row, post-target row"
docker volume create "${PREFIX}-pgdata" >/dev/null
docker volume create "${PREFIX}-repo" >/dev/null
docker run -d --name "${PREFIX}-primary" --network none \
	-e POSTGRES_USER=fvpn -e POSTGRES_DB=fvpn -e POSTGRES_HOST_AUTH_METHOD=trust \
	-e FVPN_PG_ARCHIVE_TIMEOUT=10s \
	-v "${PREFIX}-pgdata:/var/lib/postgresql/data" \
	-v "${PREFIX}-repo:/var/lib/pgbackrest" \
	"$IMAGE" >/dev/null
wait_primary "${PREFIX}-primary"
pass "primary ready archive_mode=$(docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -tAc 'SHOW archive_mode;' | tr -d '[:space:]') archive_timeout=$(docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -tAc 'SHOW archive_timeout;' | tr -d '[:space:]')"

section "handover SQL against the migrated schema"
migrate_bin="$LOG_DIR/fvpn-migrate"
(
	cd "$ROOT"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOTOOLCHAIN=local go build -trimpath -o "$migrate_bin" ./internal/db/cmd/fvpn-migrate
)
docker cp "$migrate_bin" "${PREFIX}-primary:/usr/local/bin/fvpn-migrate"
docker exec -e FVPN_DATABASE_URL='postgres://fvpn@127.0.0.1:5432/fvpn?sslmode=disable' \
	"${PREFIX}-primary" /usr/local/bin/fvpn-migrate up >"$LOG_DIR/migrate.txt" 2>&1
printf 'migrate status head:\n%s\n' "$(head -n 5 "$LOG_DIR/migrate.txt")"
if grep -q '00181' "${ROOT}/docs/handover/deploy.md"; then
	pass "deploy.md §3.3 mentions migration 00181"
else
	fail "docs/handover/deploy.md §3.3 expected status lists 19 migrations through 00180 and omits 00181 (repo has $(set -- "${ROOT}"/db/migrations/*.sql; echo "$#") migration files)"
fi
run_bad_sql() {
	local name="$1" sql="$2"
	local out="$LOG_DIR/sql-${name}.txt"
	set +e
	docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -v ON_ERROR_STOP=1 -c "$sql" >"$out" 2>&1
	local rc=$?
	set -e
	printf 'SQL %s rc=%s\n' "$name" "$rc"
	if [[ "$rc" -ne 0 ]]; then
		fail "handover query ${name} failed: $(tail -n 3 "$out" | tr '\n' ' ')"
	else
		pass "handover query ${name} ran"
	fi
}
# Original lines 331-334 hard-coded the obsolete queries. Read every SQL
# fence from the handover so this proof cannot keep testing a superseded query.
python3 - "$ROOT/docs/handover/restore-drill.md" >"$LOG_DIR/handover.sql" <<'PYSQL'
import pathlib, sys
inside = False
print("SET default_transaction_read_only=on;")
for line in pathlib.Path(sys.argv[1]).read_text().splitlines():
    if line.strip() == "```sql":
        inside = True
    elif inside and line.strip() == "```":
        inside = False
    elif inside:
        print(line)
PYSQL
run_bad_sql handover "$(cat "$LOG_DIR/handover.sql")"

stanza_out="$LOG_DIR/stanza-create.txt"
set +e
bash "${ROOT}/deploy/backup/backup.sh" --container "${PREFIX}-primary" stanza-create >"$stanza_out" 2>&1
stanza_rc=$?
set -e
printf 'stanza-create rc=%s\n' "$stanza_rc"
if [[ "$stanza_rc" -ne 0 ]]; then
	fail "backup.sh stanza-create failed on the image that auto-creates the stanza (rc=${stanza_rc})"
	tail -n 30 "$stanza_out"
else
	pass "stanza-create is idempotent"
fi

docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -v ON_ERROR_STOP=1 -c \
	"CREATE TABLE _backup_drill (marker text PRIMARY KEY, note text NOT NULL); INSERT INTO _backup_drill VALUES ('A', 'before backup');"
bash "${ROOT}/deploy/backup/backup.sh" --container "${PREFIX}-primary" full >"$LOG_DIR/full.txt" 2>&1
docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -v ON_ERROR_STOP=1 -c \
	"INSERT INTO _backup_drill VALUES ('M', 'after backup, before target');"
sleep 0.3
target="$(docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -tAc \
	"SELECT to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"');" | tr -d '[:space:]')"
sleep 0.3
docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -v ON_ERROR_STOP=1 -c \
	"INSERT INTO _backup_drill VALUES ('B', 'after target');"
bash "${ROOT}/deploy/backup/backup.sh" --container "${PREFIX}-primary" wal-switch >"$LOG_DIR/wal.txt" 2>&1
primary_markers="$(docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -tAc \
	"SELECT string_agg(marker, ',' ORDER BY marker) FROM _backup_drill;" | tr -d '[:space:]')"
printf 'primary=%s target=%s\n' "$primary_markers" "$target"
if [[ "$primary_markers" != "A,B,M" ]]; then
	fail "primary markers unexpected: ${primary_markers}"
fi

arch_lag="$(docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -tAc \
	"SELECT EXTRACT(EPOCH FROM (clock_timestamp() - last_archived_time))::bigint FROM pg_stat_archiver;" | tr -d '[:space:]')"
printf 'pg_stat_archiver lag_seconds=%s (production archive_timeout default in fvpn-archiving.conf is 300s; this container overrode 10s)\n' "$arch_lag"

section "shipped predicate versus --type=immediate (backup consistency only)"
docker volume create "${PREFIX}-immediate" >/dev/null
docker run --rm --name "${PREFIX}-immediate-init" --network none --entrypoint /bin/bash \
	-v "${PREFIX}-immediate:/var/lib/postgresql/data" \
	-v "${PREFIX}-repo:/var/lib/pgbackrest:ro" \
	"$IMAGE" -c '
		set -euo pipefail
		chown postgres:postgres /var/lib/postgresql/data
		chmod 0700 /var/lib/postgresql/data
		exec runuser -u postgres -- pgbackrest --stanza=fvpn \
			--pg1-path=/var/lib/postgresql/data --repo1-path=/var/lib/pgbackrest \
			--log-path=/tmp --type=immediate --target-action=promote restore
	' >"$LOG_DIR/immediate-restore.txt" 2>&1
docker run -d --name "${PREFIX}-immediate-pg" --network none \
	-e FVPN_PG_ISOLATED_RESTORE=1 -e POSTGRES_USER=fvpn -e POSTGRES_DB=fvpn -e POSTGRES_HOST_AUTH_METHOD=trust \
	-v "${PREFIX}-immediate:/var/lib/postgresql/data" \
	-v "${PREFIX}-repo:/var/lib/pgbackrest:ro" \
	"$IMAGE" >/dev/null
imm_ready=0
for _ in $(seq 1 90); do
	if imm="$(docker exec --user postgres "${PREFIX}-immediate-pg" psql -U fvpn -d fvpn -tAc \
		"SELECT COALESCE(string_agg(marker, ',' ORDER BY marker), '') FROM _backup_drill;" 2>/dev/null)"; then
		imm="$(printf '%s' "$imm" | tr -d '[:space:]')"
		imm_ready=1
		break
	fi
	sleep 0.5
done
printf 'immediate markers=%q ready=%s\n' "${imm:-}" "$imm_ready"
# Original lines 403-408 marked a correct --type=immediate result as a product
# failure unconditionally. Require that negative control AND the stronger drill.
if [[ "$imm_ready" -eq 1 && "$imm" == "A" ]] &&
	grep -q 'restoredMarkersCSV != "A,M"' "$ROOT/tests/backup/drill_integration_test.go" &&
	grep -q "VALUES ('M', 'after backup stop, before target')" "$ROOT/tests/backup/drill_integration_test.go"; then
	pass "immediate restore yields A only and cannot satisfy shipped A,M drill predicate"
else
	fail "immediate negative control or WAL-only drill predicate missing (markers=${imm:-unready})"
fi

section "real time restore must keep M and drop B"
time_out="$LOG_DIR/time-restore.txt"
set +e
bash "${ROOT}/deploy/backup/restore-pitr.sh" \
	--target-time "$target" \
	--isolated-container "${PREFIX}-time" \
	--target-volume "${PREFIX}-time-data" \
	--repo-volume "${PREFIX}-repo" \
	--primary-container "${PREFIX}-primary" \
	--image "$IMAGE" >"$time_out" 2>&1
time_rc=$?
set -e
if [[ "$time_rc" -ne 0 ]]; then
	fail "time restore failed rc=${time_rc}"
	tail -n 40 "$time_out"
else
	time_markers="$(docker exec --user postgres "${PREFIX}-time" psql -U fvpn -d fvpn -tAc \
		"SELECT COALESCE(string_agg(marker, ',' ORDER BY marker), '') FROM _backup_drill;" | tr -d '[:space:]')"
	printf 'time-restore markers=%s target=%s\n' "$time_markers" "$target"
	if [[ "$time_markers" == "A,M" ]]; then
		pass "restore-pitr.sh --target-time kept M and dropped B (the shipped drill requires M)"
	else
		fail "time restore markers=${time_markers}; a PITR proof requires A,M present and B absent"
	fi
fi

section "configured repository copy, archive lag, and failure propagation"
# Exercise host pgBackRest selection without touching host PostgreSQL: this
# adapter runs the configured pgBackRest in this run's primary only.
cat >"${WRAP_BIN}/pgbackrest" <<EOFPG
#!/bin/bash
exec docker exec --user postgres "${PREFIX}-primary" pgbackrest "\$@"
EOFPG
chmod 0755 "${WRAP_BIN}/pgbackrest"
bash "${ROOT}/deploy/backup/verify.sh" --host-repo --primary-container "${PREFIX}-primary" \
	--image "$IMAGE" --verify-prefix "${PREFIX}-copy" --migrate-bin "$migrate_bin" \
	--json-out "$LOG_DIR/copy.json" >"$LOG_DIR/copy-verify.txt" 2>&1 || {
	cat "$LOG_DIR/copy-verify.txt"
	fail "configured repository copy did not verify"
}
if [[ -f "$LOG_DIR/copy.json" ]] && python3 - "$LOG_DIR/copy.json" <<'PYSTATE'
import json, sys
record = json.load(open(sys.argv[1]))
assert record['status'] == 'PASS'
assert 0 <= record['archiver_lag_ms'] <= 900000
assert 0 <= record['repository_archive_lag_ms'] <= 900000
assert record['repository_newest_wal']
PYSTATE
then
	pass "host-configured repository copy restores and reports both archive ages"
else
	fail "missing or invalid repository archive evidence"
fi
# A failed verify after a successful backup must fail backup.sh for both types.
cat >"${WRAP_BIN}/psql" <<'EOFPSQL'
#!/bin/bash
printf 'on\n'
EOFPSQL
cat >"${WRAP_BIN}/pgbackrest" <<'EOFPG'
#!/bin/bash
printf '%s\n' "$*" >>"$PROOF_BACKUP_CALLS"
[[ "$*" != *verify* ]] || exit 42
EOFPG
chmod 0755 "${WRAP_BIN}/psql" "${WRAP_BIN}/pgbackrest"
export PROOF_BACKUP_CALLS="$LOG_DIR/backup-calls.txt"
for kind in full diff; do
	if bash "${ROOT}/deploy/backup/backup.sh" "$kind" >"$LOG_DIR/failure-${kind}.txt" 2>&1; then
		fail "${kind} ignored failed repository verification"
	elif grep -q -- "--type=${kind} backup" "$PROOF_BACKUP_CALLS" && grep -q verify "$PROOF_BACKUP_CALLS"; then
		pass "${kind} propagates failed repository verification"
	else
		fail "${kind} did not run both backup and verification"
	fi
done
rm -f "${WRAP_BIN}/psql" "${WRAP_BIN}/pgbackrest"

section "encrypted full/diff restore with native and compatibility cipher input"
# TEST ONLY - NOT A SECRET. The value stays in env, including docker run argv.
export PGBACKREST_REPO1_CIPHER_PASS="$CIPHER"
export PGBACKREST_REPO1_CIPHER_TYPE=aes-256-cbc
docker run -d --name "${PREFIX}-encrypted-primary" --network none \
	-e PGBACKREST_REPO1_CIPHER_PASS -e PGBACKREST_REPO1_CIPHER_TYPE \
	-e POSTGRES_USER=fvpn -e POSTGRES_DB=fvpn -e POSTGRES_HOST_AUTH_METHOD=trust \
	-v "${PREFIX}-encrypted-data:/var/lib/postgresql/data" \
	-v "${PREFIX}-encrypted-repo:/var/lib/pgbackrest" "$IMAGE" >/dev/null
wait_primary "${PREFIX}-encrypted-primary"
docker exec --user postgres "${PREFIX}-encrypted-primary" psql -U fvpn -d fvpn -c \
	"CREATE TABLE cipher_marker AS SELECT 1 AS value;" >/dev/null
for kind in full diff; do
	bash "$ROOT/deploy/backup/backup.sh" --container "${PREFIX}-encrypted-primary" "$kind" >"$LOG_DIR/encrypted-${kind}.txt" 2>&1
done
for input in native compatibility; do
	if [[ "$input" == compatibility ]]; then
		export FVPN_PGBACKREST_CIPHER_PASS="$PGBACKREST_REPO1_CIPHER_PASS"
		unset PGBACKREST_REPO1_CIPHER_PASS
	fi
	bash "$ROOT/deploy/backup/restore-pitr.sh" --latest \
		--isolated-container "${PREFIX}-encrypted-${input}" --target-volume "${PREFIX}-encrypted-${input}-data" \
		--repo-volume "${PREFIX}-encrypted-repo" --image "$IMAGE" >"$LOG_DIR/encrypted-${input}.txt" 2>&1 || {
		cat "$LOG_DIR/encrypted-${input}.txt"
		fail "encrypted ${input} restore failed"
	}
	if [[ "$(docker exec --user postgres "${PREFIX}-encrypted-${input}" psql -U fvpn -d fvpn -tAc 'SELECT value FROM cipher_marker')" == 1 ]]; then
		pass "encrypted ${input} restore recovered marker"
	else
		fail "encrypted ${input} restore lost marker"
	fi
done
unset FVPN_PGBACKREST_CIPHER_PASS PGBACKREST_REPO1_CIPHER_TYPE
if grep -F -- "$CIPHER" "$LOG_DIR/docker-argv.log" >/dev/null; then
	fail "cipher value appeared anywhere on Docker argv"
else
	pass "encrypted restores keep cipher value off every Docker argv"
fi

section "documented raw pgbackrest restore targets the live PGDATA"
docker exec --user postgres "${PREFIX}-primary" bash -c 'echo live-only > /var/lib/postgresql/data/LIVE_ONLY_FILE'
docker cp "${PREFIX}-primary:/etc/pgbackrest/pgbackrest.conf" "$LOG_DIR/pgbackrest.conf"
chmod a+r "$LOG_DIR/pgbackrest.conf"
pg1="$(awk -F= '/^pg1-path=/{print $2}' "$LOG_DIR/pgbackrest.conf")"
printf 'rendered pg1-path=%s\n' "$pg1"
if [[ "$pg1" == "/var/lib/postgresql/data" ]]; then
	pass "image config pg1-path is the container data directory (the path a bare pgbackrest restore uses)"
else
	fail "unexpected pg1-path ${pg1}"
fi
docker stop "${PREFIX}-primary" >/dev/null
inplace_out="$LOG_DIR/inplace.txt"
set +e
pg_target="${target/T/ }"
pg_target="${pg_target%Z}+00"
printf 'handover-shaped target=%s\n' "$pg_target"
docker run --rm --name "${PREFIX}-inplace" --network none --user postgres --entrypoint pgbackrest \
	-v "${PREFIX}-pgdata:/var/lib/postgresql/data" \
	-v "${PREFIX}-repo:/var/lib/pgbackrest" \
	-v "$LOG_DIR/pgbackrest.conf:/etc/pgbackrest/pgbackrest.conf:ro" \
	"$IMAGE" \
	--stanza=fvpn --type=time "--target=${pg_target}" --target-action=promote restore >"$inplace_out" 2>&1
inplace_rc=$?
set -e
# The handover writes "--target=2026-10-29 18:02:00+00". Match that shape from our captured timestamp.
printf 'inplace rc=%s\n' "$inplace_rc"
tail -n 15 "$inplace_out"
live_after="$(docker run --rm --name "${PREFIX}-live-read" --network none --user postgres --entrypoint cat \
	-v "${PREFIX}-pgdata:/var/lib/postgresql/data:ro" "$IMAGE" /var/lib/postgresql/data/LIVE_ONLY_FILE 2>/dev/null || true)"
if [[ "$inplace_rc" -eq 40 && "$live_after" == "live-only" ]]; then
	pass "image config (archive-async=n, no delta) refused a non-empty PGDATA (rc=40) and left LIVE_ONLY_FILE"
elif [[ "$inplace_rc" -eq 0 && -z "$live_after" ]]; then
	fail "raw pgbackrest restore (no --pg1-path, no restore-pitr.sh) removed LIVE_ONLY_FILE from the live PGDATA"
else
	fail "raw pgbackrest restore from restore-drill.md Phase 2.1 exited ${inplace_rc} live='${live_after}'"
fi

# Negative control: the old pgbackrest.conf.example set delta=y. pgBackRest's rc=40 hint is "try using --delta".
delta_conf="$LOG_DIR/pgbackrest-delta.conf"
cp "$LOG_DIR/pgbackrest.conf" "$delta_conf"
chmod a+r "$delta_conf"
if ! grep -q '^delta=' "$delta_conf"; then
	sed -i '/^\[global\]/a delta=y' "$delta_conf"
fi
docker run --rm --name "${PREFIX}-live-read" --network none --user postgres --entrypoint bash \
	-v "${PREFIX}-pgdata:/var/lib/postgresql/data" \
	"$IMAGE" -c 'echo live-only > /var/lib/postgresql/data/LIVE_ONLY_FILE' >/dev/null
delta_out="$LOG_DIR/inplace-delta.txt"
set +e
docker run --rm --name "${PREFIX}-inplace" --network none --user postgres --entrypoint pgbackrest \
	-v "${PREFIX}-pgdata:/var/lib/postgresql/data" \
	-v "${PREFIX}-repo:/var/lib/pgbackrest" \
	-v "$delta_conf:/etc/pgbackrest/pgbackrest.conf:ro" \
	"$IMAGE" \
	--stanza=fvpn --type=time "--target=${pg_target}" --target-action=promote restore >"$delta_out" 2>&1
delta_rc=$?
set -e
live_delta="$(docker run --rm --name "${PREFIX}-live-read" --network none --user postgres --entrypoint cat \
	-v "${PREFIX}-pgdata:/var/lib/postgresql/data:ro" "$IMAGE" /var/lib/postgresql/data/LIVE_ONLY_FILE 2>/dev/null || true)"
printf 'delta-restore rc=%s live=%q\n' "$delta_rc" "$live_delta"
tail -n 8 "$delta_out" || true
# Original lines 481-482 inject, and 501-506 unconditionally fail for, delta=y. Keep that dangerous
# behavior as a negative control and require it absent from production config.
if [[ "$delta_rc" -eq 0 && -z "$live_delta" ]] &&
	! grep -q '^delta=y' "$ROOT/deploy/backup/pgbackrest.conf.example" &&
	grep -q 'restore-pitr.sh' "$ROOT/docs/handover/restore-drill.md" &&
	! grep -q 'pgbackrest --stanza=fvpn --type=time' "$ROOT/docs/handover/restore-drill.md"; then
	pass "injected delta=y destroys the marker; production config and handover exclude that path"
else
	fail "delta negative control or safe handover/config invariant failed (rc=${delta_rc})"
fi

section "verify.sh without an archive-state source"
verify_out="$LOG_DIR/verify.txt"
set +e
bash "${ROOT}/deploy/backup/verify.sh" \
	--stanza fvpn \
	--repo-volume "${PREFIX}-repo" \
	--image "$IMAGE" \
	--verify-prefix "${PREFIX}-verify" \
	--migrate-bin "$migrate_bin" \
	--json-out "$LOG_DIR/verify.json" >"$verify_out" 2>&1
verify_rc=$?
set -e
printf 'verify rc=%s\n' "$verify_rc"
grep -E 'rpo_ms|status=' "$verify_out" || true
if [[ "$verify_rc" -eq 0 ]] || grep -q 'status=PASS' "$verify_out" || ! grep -q 'archive state unknown' "$verify_out"; then
	fail "verify.sh --repo-volume only (systemd fvpn-backup-verify.service) printed status=PASS and rpo_ms=0"
else
	pass "verify.sh did not report a zero-lag PASS without a primary (rc=${verify_rc})"
fi

printf '\n%s failing proof(s).\n' "$FAILS"
if [[ "$FAILS" -gt 0 ]]; then
	exit 1
fi
exit 0
