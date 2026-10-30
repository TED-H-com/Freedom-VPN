#!/usr/bin/env bash
# Adversarial proofs for the PR #86 backup / PITR deliverable.
# TODO(review): this script is expected to exit 1 on the current tree.
# It prints FAIL for each defect and PASS for each check that held.
# Containers, volumes and images it creates are named fvpn-r5-backups-review-*.
# It never starts, stops, removes or execs any other container.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
PREFIX="fvpn-r5-backups-review"
IMAGE="${FVPN_R5_IMAGE:-${PREFIX}-postgres:17}"
SRC_IMAGE="fvpn-w4-ops-backups-postgres:17"
CIPHER="review-cipher-not-a-secret"
LOG_DIR="${TMPDIR:-/tmp}/${PREFIX}-logs"
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
	rm -rf "$LIVE_TMP" "$WRAP_BIN"
}
trap cleanup EXIT

docker_real="$(command -v docker)"
WRAP_BIN="$(mktemp -d /tmp/${PREFIX}-docker-wrap.XXXXXX)"
cat >"${WRAP_BIN}/docker" <<EOF
#!/bin/bash
printf '%s\n' "\$*" >> "${LOG_DIR}/docker-argv.log"
exec ${docker_real} "\$@"
EOF
chmod 0755 "${WRAP_BIN}/docker"
export PATH="${WRAP_BIN}:${PATH}"

section "static: secrets are not echoed, but the cipher is placed on argv"
if grep -RInE 'echo .*(PGBACKREST_REPO1_S3_KEY|PGBACKREST_REPO1_CIPHER_PASS|CIPHER_PASS)' \
	"${ROOT}/deploy/backup" "${ROOT}/.github/workflows/backup.yml" \
	--exclude-dir=test >/tmp/${PREFIX}-echo-grep.txt; then
	fail "script echoes a secret env var: $(cat /tmp/${PREFIX}-echo-grep.txt)"
else
	pass "no echo of cipher/S3 env vars in deploy/backup scripts or backup.yml"
fi
if grep -n 'repo1-cipher-pass=' "${ROOT}/deploy/backup/restore-pitr.sh" | grep -q 'CIPHER_PASS'; then
	# TODO(review): passphrase is a process argument, not only an env var.
	fail "restore-pitr.sh puts the cipher passphrase on the pgbackrest/docker argument list (restore-pitr.sh:251)"
else
	pass "cipher passphrase stays off the argument list"
fi
if grep -n 'archive-async=y' "${ROOT}/deploy/backup/pgbackrest.conf.example" >/dev/null; then
	# TODO(review): production profile acks WAL before it reaches the repo.
	fail "production pgbackrest.conf.example sets archive-async=y (WAL spool success is reported to PostgreSQL before the off-host push)"
else
	pass "production profile does not enable archive-async"
fi
if grep -RInE 'pgbackrest[[:space:]]+verify|pgbackrest[[:space:]].*[[:space:]]verify([[:space:]]|$)' \
	"${ROOT}/deploy/backup" "${ROOT}/.github/workflows/backup.yml" "${ROOT}/mk/backup.mk" \
	--exclude-dir=test >/dev/null; then
	pass "a pgbackrest verify step is wired into the backup path"
else
	# TODO(review): neither the backup command nor the workflow verifies the repo.
	fail "backup.sh, verify.sh, backup.yml and backup.mk never run 'pgbackrest verify'"
fi
if grep -n 'continue-on-error' "${ROOT}/.github/workflows/backup.yml" >/dev/null; then
	fail "backup.yml sets continue-on-error"
else
	pass "backup.yml has no continue-on-error"
fi
if grep -n 'time.Since(walSwitchStart)' "${ROOT}/tests/backup/drill_integration_test.go" >/dev/null; then
	# TODO(review): the published RPO is a stopwatch around wal-switch, not archive lag.
	fail "TestAT30_BackupPITRDrill records measured_rpo_archive_lag_ms as time.Since(wal-switch), not pg_stat_archiver lag or archive_timeout"
else
	pass "drill RPO is not a wal-switch stopwatch"
fi
if grep -n 'COALESCE(last_archived_time, clock_timestamp())' "${ROOT}/deploy/backup/verify.sh" >/dev/null \
	&& grep -n 'RPO_MS=0' "${ROOT}/deploy/backup/verify.sh" >/dev/null; then
	# TODO(review): never-archived and "no primary passed" both report zero lag.
	fail "verify.sh reports rpo_ms=0 when no primary is given, and treats a NULL last_archived_time as zero lag"
else
	pass "verify.sh archive-lag measurement does not collapse to zero"
fi
if grep -n 'repo1-retention-diff=14' "${ROOT}/deploy/backup/pgbackrest.conf.example" >/dev/null \
	&& grep -n '14 days of differential' "${ROOT}/docs/ops/backups.md" >/dev/null; then
	# TODO(review): repo-retention-diff is a count, and the doc calls it days.
	fail "docs/ops/backups.md calls repo1-retention-diff=14 a 14-day differential window; pgBackRest interprets it as a count of 14 diffs"
else
	pass "differential retention units match the docs"
fi
if grep -n 'repo1-retention-archive=35' "${ROOT}/deploy/backup/pgbackrest.conf.example" >/dev/null \
	&& grep -n 'repo1-retention-archive-type=full' "${ROOT}/deploy/backup/pgbackrest.conf.example" >/dev/null \
	&& grep -n 'OnCalendar=Sun' "${ROOT}/deploy/backup/systemd/fvpn-backup-full.timer" >/dev/null; then
	# TODO(review): archive retention counts full backups, while fulls are weekly and time-expired at 35 days.
	fail "production archive retention keeps WAL for 35 full backups while full backups are weekly and expired by time at 35 days, so the 35-full WAL rule is not a 35-day expiry"
else
	pass "archive retention matches the documented 35-day window"
fi
if ! grep -n -- '--primary-container' "${ROOT}/deploy/backup/systemd/fvpn-backup-verify.service" >/dev/null; then
	# TODO(review): the installed verify unit always takes the rpo_ms=0 branch.
	fail "fvpn-backup-verify.service does not pass --primary-container, so its published rpo_ms is the zero default"
else
	pass "verify unit measures a primary"
fi
if grep -n 'pgbackrest --stanza=fvpn --type=time' "${ROOT}/docs/handover/restore-drill.md" >/dev/null \
	&& ! grep -n 'restore-pitr.sh' "${ROOT}/docs/handover/restore-drill.md" >/dev/null; then
	# TODO(review): the handover restore command targets pg1-path from the host config.
	fail "docs/handover/restore-drill.md Phase 2.1 runs raw pgbackrest restore with no --pg1-path and does not call restore-pitr.sh"
else
	pass "handover PITR command is the guarded restore script"
fi

section "static: handover SQL names objects the migrations do not create"
sql_missing=0
for obj in billing_subscriptions dns_resolver_states; do
	if grep -RIn -- "$obj" "${ROOT}/db/migrations" >/dev/null; then
		pass "migration defines ${obj}"
	else
		sql_missing=1
		fail "docs/handover/restore-drill.md queries ${obj}, which no migration creates"
	fi
done
if grep -RIn 'missing_ranges' "${ROOT}/db/migrations" >/dev/null; then
	pass "usage_checkpoints.missing_ranges is a real column"
else
	sql_missing=1
	fail "docs/handover/restore-drill.md selects usage_checkpoints.missing_ranges; migrations never add that column (it exists only on the API checkpoint payload)"
fi
if grep -n 'FROM operations' "${ROOT}/docs/handover/restore-drill.md" | grep -q .; then
	if grep -n 'target_resource' "${ROOT}/db/migrations/"*.sql >/dev/null; then
		pass "operations.target_resource exists"
	else
		fail "docs/handover/restore-drill.md selects operations.state and operations.target_resource; the table has status and no target_resource"
	fi
fi

section "guard: same directory and postmaster.pid are refused; a child directory is not"
LIVE_TMP="$(mktemp -d /tmp/${PREFIX}-live.XXXXXX)"
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
	# TODO(review): guard compares paths for equality only.
	fail "restore-pitr.sh created ${child} inside the live data directory before invoking pgbackrest (rc=${child_rc})"
	printf 'child stderr:\n%s\n' "$(cat "$child_out")"
else
	pass "child of the live data directory was refused (rc=${child_rc})"
fi

if ! docker image inspect "$SRC_IMAGE" >/dev/null 2>&1; then
	section "build review postgres image"
	docker build -t "$IMAGE" "${ROOT}/deploy/backup/postgres" >"$LOG_DIR/docker-build.txt" 2>&1
else
	docker tag "$SRC_IMAGE" "$IMAGE"
	pass "tagged existing ${SRC_IMAGE} as ${IMAGE} (no container created)"
fi

section "RPO SQL: NULL last_archived_time is reported as zero lag"
docker run -d --name "${PREFIX}-nullrpo" --network none \
	-e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_USER=fvpn -e POSTGRES_DB=fvpn \
	postgres:17 >"$LOG_DIR/nullrpo-id.txt"
for _ in $(seq 1 60); do
	if docker exec --user postgres "${PREFIX}-nullrpo" pg_isready -U fvpn -d fvpn >/dev/null 2>&1; then
		break
	fi
	sleep 0.5
done
rpo_sql="SELECT COALESCE(GREATEST(0, (EXTRACT(EPOCH FROM (clock_timestamp() - COALESCE(last_archived_time, clock_timestamp()))) * 1000)::bigint), 0) FROM pg_stat_archiver;"
null_rpo="$(docker exec --user postgres "${PREFIX}-nullrpo" psql -U fvpn -d fvpn -tAc "$rpo_sql" | tr -d '[:space:]')"
null_raw="$(docker exec --user postgres "${PREFIX}-nullrpo" psql -U fvpn -d fvpn -tAc "SELECT archive_mode, last_archived_time IS NULL, archived_count FROM pg_stat_archiver, (SELECT current_setting('archive_mode') AS archive_mode) s;" | tr -d '[:space:]')"
printf 'null-archive query -> rpo_ms=%s detail=%s\n' "$null_rpo" "$null_raw"
if [[ "$null_rpo" == "0" ]]; then
	# TODO(review): a primary that has never archived looks fully caught up.
	fail "verify.sh lag SQL returned 0 on a cluster whose last_archived_time is NULL (${null_raw})"
else
	pass "NULL last_archived_time did not report zero lag (${null_rpo})"
fi
docker rm -f "${PREFIX}-nullrpo" >/dev/null

section "volume guard: stopped container is held; removed container's volume is wiped"
docker volume create "${PREFIX}-held" >/dev/null
docker volume create "${PREFIX}-orphan" >/dev/null
docker volume create "${PREFIX}-empty-repo" >/dev/null
docker run -d --name "${PREFIX}-heldsrc" --network none \
	-v "${PREFIX}-held:/var/lib/postgresql/data" \
	-e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_USER=fvpn -e POSTGRES_DB=fvpn \
	postgres:17 >/dev/null
for _ in $(seq 1 60); do
	if docker exec --user postgres "${PREFIX}-heldsrc" pg_isready -U fvpn -d fvpn >/dev/null 2>&1; then
		break
	fi
	sleep 0.5
done
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
if [[ -z "$marker_after" ]]; then
	# TODO(review): in-use check ignores volumes whose container object is gone, then find -delete wipes them.
	fail "restore-pitr.sh deleted LIVE_MARKER from an unmounted volume that still held a database (rc=${orphan_rc})"
else
	pass "unmounted volume contents survived (marker=${marker_after})"
fi
if grep -F -- "--repo1-cipher-pass=${CIPHER}" "$LOG_DIR/docker-argv.log" >/dev/null; then
	# TODO(review): docker run argv contains the passphrase.
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
ready=0
for _ in $(seq 1 90); do
	if docker exec --user postgres "${PREFIX}-primary" psql -U fvpn -d fvpn -tAc 'SELECT 1' 2>/dev/null | grep -q 1; then
		ready=1
		break
	fi
	sleep 0.5
done
if [[ "$ready" -ne 1 ]]; then
	docker logs "${PREFIX}-primary" >"$LOG_DIR/primary-logs.txt" 2>&1 || true
	fail "primary did not become ready"
	printf '%s\n' "$(tail -n 40 "$LOG_DIR/primary-logs.txt")"
	printf '\n%s failing proof(s)\n' "$FAILS"
	exit 1
fi
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
	# TODO(review): twenty migration files exist; the handover status block stops at 00180.
	fail "docs/handover/deploy.md §3.3 expected status lists 19 migrations through 00180 and omits 00181 (repo has $(ls "${ROOT}/db/migrations/"*.sql | wc -l) migration files)"
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
run_bad_sql billing_subscriptions "SELECT id, member_id, provider_subscription_id, status, current_period_end, updated_at FROM billing_subscriptions ORDER BY updated_at DESC LIMIT 50;"
run_bad_sql operations_state "SELECT id, kind, state, target_resource, error_code, updated_at FROM operations WHERE state IN ('pending', 'running', 'failed', 'partially_succeeded') ORDER BY updated_at DESC;"
run_bad_sql missing_ranges "SELECT node_id, incarnation_id, ledger_epoch, highest_contiguous_seq, max_seen_seq, missing_ranges, updated_at FROM usage_checkpoints ORDER BY updated_at DESC;"
run_bad_sql dns_resolver_states "SELECT resolver_id, applied_generation, last_success_at, last_error_code FROM dns_resolver_states;"

stanza_out="$LOG_DIR/stanza-create.txt"
set +e
bash "${ROOT}/deploy/backup/backup.sh" --container "${PREFIX}-primary" stanza-create >"$stanza_out" 2>&1
stanza_rc=$?
set -e
printf 'stanza-create rc=%s\n' "$stanza_rc"
if [[ "$stanza_rc" -ne 0 ]]; then
	# TODO(review): the image init script already created the stanza; the drill calls stanza-create again.
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
		find /var/lib/postgresql/data -mindepth 1 -delete
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
if [[ "$imm_ready" -eq 1 && "$imm" == "A" ]]; then
	# TODO(review): shipped drill asserts restored markers == A, which this non-PITR restore also satisfies.
	fail "a --type=immediate restore (no WAL past the base backup) has markers '${imm}'. The shipped drill only checks that A is present and B is absent, so it passes without proving that post-backup pre-target row M was recovered"
else
	pass "immediate restore did not satisfy the shipped A-and-not-B predicate (markers=${imm:-unready})"
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
		pass "restore-pitr.sh --target-time kept M and dropped B (the shipped drill never inserts M)"
	else
		# TODO(review): time target did not recover the post-backup pre-target row, or it kept the post-target row.
		fail "time restore markers=${time_markers}; a PITR proof requires A,M present and B absent"
	fi
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
	# TODO(review): the command in restore-drill.md Phase 2.1 replaced the live data directory.
	fail "raw pgbackrest restore (no --pg1-path, no restore-pitr.sh) removed LIVE_ONLY_FILE from the live PGDATA"
else
	# TODO(review): the documented command did not match the rc=40 refusal seen with pgBackRest 2.60.0.
	fail "raw pgbackrest restore from restore-drill.md Phase 2.1 exited ${inplace_rc} live='${live_after}'"
fi

# pgbackrest.conf.example sets delta=y. pgBackRest's rc=40 hint is "try using --delta".
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
if [[ "$delta_rc" -eq 0 && -z "$live_delta" ]]; then
	# TODO(review): the example production profile makes the handover command rewrite the live data directory.
	fail "handover pgbackrest command with delta=y (deploy/backup/pgbackrest.conf.example) restored over the live PGDATA and removed LIVE_ONLY_FILE"
else
	pass "delta=y restore left LIVE_ONLY_FILE in place (rc=${delta_rc})"
fi

section "verify.sh without --primary-container (the systemd unit)"
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
if [[ "$verify_rc" -eq 0 ]] && grep -q 'rpo_ms=0' "$verify_out" && grep -q 'status=PASS' "$verify_out"; then
	# TODO(review): PASS plus rpo_ms=0 is what the daily unit records.
	fail "verify.sh --repo-volume only (systemd fvpn-backup-verify.service) printed status=PASS and rpo_ms=0"
else
	pass "verify.sh did not report a zero-lag PASS without a primary (rc=${verify_rc})"
fi

printf '\n%s failing proof(s). TODO(review): exit status is non-zero while these defects remain.\n' "$FAILS"
if [[ "$FAILS" -gt 0 ]]; then
	exit 1
fi
exit 0
