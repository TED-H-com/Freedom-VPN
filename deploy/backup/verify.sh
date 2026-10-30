#!/usr/bin/env bash
# Freedom VPN Backup Verification & RPO/RTO Measurement Script (`deploy/backup/verify.sh`)
# Spec references: §15.4, §17.2, §17.3, §17.4, AT30 (SECOPS-05)
#
# Restores the latest backup (or specified target time) into a throwaway
# `--network none` container, verifies schema migration status via `fvpn-migrate status`,
# runs the read-only post-restore checklist (`post-restore-checklist.sql`),
# prints measured RPO/RTO, and cleans up all temporary Docker resources on exit.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

STANZA="${FVPN_PGBACKREST_STANZA:-fvpn}"
REPO_VOLUME=""
IMAGE="${FVPN_BACKUP_IMAGE:-fvpn-w4-ops-backups-postgres:17}"
PRIMARY_CONTAINER="${FVPN_PRIMARY_CONTAINER:-}"
VERIFY_PREFIX="fvpn-w4-ops-backups-verify-$$"
DB_USER="${FVPN_BACKUP_DB_USER:-fvpn}"
DB_NAME="${FVPN_BACKUP_DB_NAME:-fvpn}"
TARGET_TIME=""
MIGRATE_BIN=""
JSON_OUT=""

usage() {
	cat >&2 <<'EOF'
Usage:
  verify.sh --repo-volume <volume> [options]

Options:
  --repo-volume <volume>        pgBackRest repository Docker volume or host path
  --primary-container <name>    Optional primary container to measure live WAL archive lag (RPO)
  --image <image>               PostgreSQL + pgBackRest image (default: fvpn-w4-ops-backups-postgres:17)
  --stanza <name>               pgBackRest stanza name (default: fvpn)
  --target-time <timestamp>     Optional UTC timestamp for PITR verification (default: latest)
  --verify-prefix <prefix>      Container/volume prefix (must start with fvpn-w4-ops-backups-)
  --db-user <user>              PostgreSQL user (default: fvpn)
  --db-name <db>                PostgreSQL database (default: fvpn)
  --migrate-bin <path>          Optional prebuilt fvpn-migrate linux/amd64 binary
  --json-out <path>             Optional file path to write JSON measurement summary
  -h, --help                    Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--repo-volume)
		REPO_VOLUME="${2:-}"
		shift 2
		;;
	--primary-container)
		PRIMARY_CONTAINER="${2:-}"
		shift 2
		;;
	--image)
		IMAGE="${2:-}"
		shift 2
		;;
	--stanza)
		STANZA="${2:-}"
		shift 2
		;;
	--target-time)
		TARGET_TIME="${2:-}"
		shift 2
		;;
	--verify-prefix)
		VERIFY_PREFIX="${2:-}"
		shift 2
		;;
	--db-user)
		DB_USER="${2:-}"
		shift 2
		;;
	--db-name)
		DB_NAME="${2:-}"
		shift 2
		;;
	--migrate-bin)
		MIGRATE_BIN="${2:-}"
		shift 2
		;;
	--json-out)
		JSON_OUT="${2:-}"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "verify.sh: unknown argument: $1" >&2
		usage
		exit 2
		;;
	esac
done

if [[ -z "$REPO_VOLUME" ]]; then
	echo "verify.sh: --repo-volume is required" >&2
	usage
	exit 2
fi

VERIFY_CONTAINER="${VERIFY_PREFIX}-pg"
VERIFY_VOLUME="${VERIFY_PREFIX}-pgdata"
WORK_TMP=""

cleanup() {
	docker rm -fv "$VERIFY_CONTAINER" "${VERIFY_CONTAINER}-init" >/dev/null 2>&1 || true
	docker volume rm -f "$VERIFY_VOLUME" >/dev/null 2>&1 || true
	if [[ -n "$WORK_TMP" ]] && [[ -d "$WORK_TMP" ]]; then
		rm -rf "$WORK_TMP"
	fi
}
trap cleanup EXIT

mkdir -p "${REPO_ROOT}/.agent/tmp"
WORK_TMP="$(mktemp -d "${REPO_ROOT}/.agent/tmp/backup-verify.XXXXXX")"
export TMPDIR="$WORK_TMP" GOTMPDIR="$WORK_TMP"

# Build fvpn-migrate if not supplied by caller.
if [[ -z "$MIGRATE_BIN" ]]; then
	MIGRATE_BIN="${WORK_TMP}/fvpn-migrate"
	(
		cd "$REPO_ROOT"
		GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
			go build -trimpath -o "$MIGRATE_BIN" ./internal/db/cmd/fvpn-migrate
	)
fi

# Measure RPO (archive lag in milliseconds) from primary pg_stat_archiver if primary container is given.
RPO_MS=0
if [[ -n "$PRIMARY_CONTAINER" ]]; then
	RPO_MS="$(docker exec --user postgres "$PRIMARY_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc "
		SELECT COALESCE(
			GREATEST(0, (EXTRACT(EPOCH FROM (clock_timestamp() - COALESCE(last_archived_time, clock_timestamp()))) * 1000)::bigint),
			0
		)
		FROM pg_stat_archiver;
	" | tr -d '[:space:]')"
fi

RTO_START_NS="$(date +%s%N)"

RESTORE_ARGS=(
	--stanza "$STANZA"
	--isolated-container "$VERIFY_CONTAINER"
	--target-volume "$VERIFY_VOLUME"
	--repo-volume "$REPO_VOLUME"
	--image "$IMAGE"
	--db-user "$DB_USER"
	--db-name "$DB_NAME"
)
if [[ -n "$PRIMARY_CONTAINER" ]]; then
	RESTORE_ARGS+=(--primary-container "$PRIMARY_CONTAINER")
fi
if [[ -n "$TARGET_TIME" ]]; then
	RESTORE_ARGS+=(--target-time "$TARGET_TIME")
else
	RESTORE_ARGS+=(--latest)
fi

"${SCRIPT_DIR}/restore-pitr.sh" "${RESTORE_ARGS[@]}"

# Copy fvpn-migrate into the isolated network-none container and verify migration status.
docker cp "$MIGRATE_BIN" "${VERIFY_CONTAINER}:/usr/local/bin/fvpn-migrate"
MIGRATE_STATUS_OUT="$(docker exec \
	-e "FVPN_DATABASE_URL=postgres://${DB_USER}@127.0.0.1:5432/${DB_NAME}?sslmode=disable" \
	"$VERIFY_CONTAINER" /usr/local/bin/fvpn-migrate status)"

APPLIED_COUNT="$(printf '%s\n' "$MIGRATE_STATUS_OUT" | grep -c ' applied$' || true)"
PENDING_COUNT="$(printf '%s\n' "$MIGRATE_STATUS_OUT" | grep -c ' pending$' || true)"
if [[ "$APPLIED_COUNT" -eq 0 ]] || [[ "$PENDING_COUNT" -ne 0 ]]; then
	echo "verify.sh: migration verification failed (applied=${APPLIED_COUNT}, pending=${PENDING_COUNT}):" >&2
	printf '%s\n' "$MIGRATE_STATUS_OUT" >&2
	exit 1
fi

# Run the read-only post-restore checklist against the restored database.
CHECKLIST_ARGS=(-v ON_ERROR_STOP=1)
if [[ -n "$TARGET_TIME" ]]; then
	CHECKLIST_ARGS+=(-v "restore_point=${TARGET_TIME}")
fi
CHECKLIST_OUT="$(docker exec -i --user postgres "$VERIFY_CONTAINER" \
	psql -U "$DB_USER" -d "$DB_NAME" "${CHECKLIST_ARGS[@]}" <"${SCRIPT_DIR}/post-restore-checklist.sql")"

RTO_END_NS="$(date +%s%N)"
RTO_MS=$(((RTO_END_NS - RTO_START_NS) / 1000000))

cat <<EOF
=== Freedom VPN Backup Restore Verification Summary ===
verify_container=${VERIFY_CONTAINER}
network_mode=none
migrations_applied=${APPLIED_COUNT}
migrations_pending=${PENDING_COUNT}
rpo_ms=${RPO_MS}
rto_ms=${RTO_MS}
status=PASS
EOF

if [[ -n "$JSON_OUT" ]]; then
	cat >"$JSON_OUT" <<EOF
{
  "verify_container": "${VERIFY_CONTAINER}",
  "network_mode": "none",
  "migrations_applied": ${APPLIED_COUNT},
  "migrations_pending": ${PENDING_COUNT},
  "rpo_ms": ${RPO_MS},
  "rto_ms": ${RTO_MS},
  "status": "PASS"
}
EOF
fi

if [[ "${FVPN_VERIFY_PRINT_CHECKLIST:-0}" == "1" ]]; then
	printf '%s\n' "$CHECKLIST_OUT"
fi
