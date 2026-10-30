#!/usr/bin/env bash
# Freedom VPN Backup Verification & RPO/RTO Measurement Script (`deploy/backup/verify.sh`)
# Spec references: §15.4, §17.2, §17.3, §17.4, AT30 (SECOPS-05)
#
# Restores the latest backup (or specified target time) into a throwaway
# `--network none` container, verifies schema migration status via `fvpn-migrate status`,
# runs the read-only post-restore checklist (`post-restore-checklist.sql`),
# prints archive ages and local restore/verification duration, and cleans up all temporary Docker resources on exit.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

STANZA="${FVPN_PGBACKREST_STANZA:-fvpn}"
REPO_VOLUME=""
IMAGE="${FVPN_BACKUP_IMAGE:-fvpn-w4-ops-backups-postgres:17}"
PRIMARY_CONTAINER="${FVPN_PRIMARY_CONTAINER:-}"
PRIMARY_HOST=0
HOST_REPO=0
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
  --primary-container <name>    Primary container for live archive state
  --primary-host                Read archive state using host psql (local postgres OS user)
  --host-repo                   Copy repo 1 using host pgBackRest config/env (including S3)
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
	--host-repo)
		HOST_REPO=1
		shift
		;;
	--primary-host)
		PRIMARY_HOST=1
		shift
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

if [[ ! "$VERIFY_PREFIX" =~ ^fvpn-w4-ops-backups-[a-z0-9]+(-[a-z0-9]+)*$ || ${#VERIFY_PREFIX} -gt 70 ]]; then
	echo "verify.sh: invalid --verify-prefix family or run suffix" >&2
	exit 2
fi
if [[ ! "$STANZA" =~ ^[a-zA-Z0-9_-]+$ ]]; then
	echo "verify.sh: invalid stanza" >&2
	exit 2
fi
if { [[ -z "$PRIMARY_CONTAINER" ]] && [[ "$PRIMARY_HOST" == 0 ]]; } ||
	{ [[ -n "$PRIMARY_CONTAINER" ]] && [[ "$PRIMARY_HOST" == 1 ]]; }; then
	echo "verify.sh: archive state unknown; select --primary-container or --primary-host" >&2
	exit 2
fi
if { [[ -z "$REPO_VOLUME" ]] && [[ "$HOST_REPO" == 0 ]]; } ||
	{ [[ -n "$REPO_VOLUME" ]] && [[ "$HOST_REPO" == 1 ]]; }; then
	echo "verify.sh: select --repo-volume or --host-repo" >&2
	exit 2
fi

VERIFY_CONTAINER="${VERIFY_PREFIX}-pg"
VERIFY_VOLUME="${VERIFY_PREFIX}-pgdata"
# Refuse pre-existing resources before registering any destructive cleanup.
for candidate in "$VERIFY_CONTAINER" "${VERIFY_CONTAINER}-init"; do
	if docker container inspect "$candidate" >/dev/null 2>&1; then
		echo "verify.sh: refusing existing verification container" >&2
		exit 2
	fi
done
if docker volume inspect "$VERIFY_VOLUME" >/dev/null 2>&1; then
	echo "verify.sh: refusing existing verification volume" >&2
	exit 2
fi
WORK_TMP=""

cleanup() {
	docker rm -fv "$VERIFY_CONTAINER" "${VERIFY_CONTAINER}-init" >/dev/null 2>&1 || true
	docker volume rm -f "$VERIFY_VOLUME" >/dev/null 2>&1 || true
	if [[ -n "$WORK_TMP" ]] && [[ -d "$WORK_TMP" ]]; then
		rm -rf "$WORK_TMP"
	fi
}
trap cleanup EXIT

SCRATCH_ROOT="${FVPN_BACKUP_SCRATCH_ROOT:-${REPO_ROOT}/.agent/tmp}"
mkdir -p "$SCRATCH_ROOT"
WORK_TMP="$(mktemp -d "${SCRATCH_ROOT}/backup-verify.XXXXXX")"
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

# Unknown, disabled, future, or stale archive state fails before any PASS.
if [[ "$PRIMARY_HOST" == 1 ]]; then
	PRIMARY_PSQL=(psql -U "$DB_USER" -d "$DB_NAME")
	if [[ "$(id -u)" == 0 ]]; then
		PRIMARY_PSQL=(runuser -u postgres -- "${PRIMARY_PSQL[@]}")
	fi
else
	PRIMARY_PSQL=(docker exec -i --user postgres "$PRIMARY_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME")
fi
ARCHIVER_LAG_MS="$("${PRIMARY_PSQL[@]}" -v ON_ERROR_STOP=1 -tA <"${SCRIPT_DIR}/archive-state.sql" | tr -d '[:space:]')"
if [[ ! "$ARCHIVER_LAG_MS" =~ ^[0-9]+$ ]] || (( ARCHIVER_LAG_MS > 900000 )); then
	echo "verify.sh: archive state unknown, future-dated, or older than 15 minutes" >&2
	exit 1
fi

# Production uses precisely the host pgBackRest repo 1 configuration/environment.
# Copy raw encrypted objects to disk before launching network-none PostgreSQL.
if [[ "$HOST_REPO" == 1 ]]; then
	python3 "${SCRIPT_DIR}/repo-copy.py" "$STANZA" "$WORK_TMP/repository"
	chmod 0755 "$WORK_TMP"
	REPO_VOLUME="$WORK_TMP/repository"
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

# Read and checksum the repository actually mounted by the restored instance.
REPO_LIST="$(docker exec --user postgres "$VERIFY_CONTAINER" pgbackrest --stanza="$STANZA" --repo=1 --recurse --output=json repo-ls "archive/$STANZA")"
REPO_STATE="$(printf '%s' "$REPO_LIST" | python3 -c '
import json, re, sys, time
files = [(v["time"], k) for k, v in json.load(sys.stdin).items()
         if v["type"] == "file" and re.fullmatch(r"[0-9A-F]{24}-[0-9a-f]{40}(?:\..+)?", k.rsplit("/", 1)[-1])]
if not files:
    sys.exit("verify.sh: repository archive state unknown (no WAL)")
stamp, name = max(files)
lag = int((time.time() - stamp) * 1000)
if lag < 0 or lag > 900000:
    sys.exit("verify.sh: repository WAL is future-dated or older than 15 minutes")
print(str(lag) + " " + name)
')"
read -r REPOSITORY_LAG_MS REPOSITORY_WAL <<<"$REPO_STATE"
docker exec --user postgres "$VERIFY_CONTAINER" pgbackrest --stanza="$STANZA" verify

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
archiver_lag_ms=${ARCHIVER_LAG_MS}
repository_archive_lag_ms=${REPOSITORY_LAG_MS}
repository_newest_wal=${REPOSITORY_WAL}
local_restore_verify_ms=${RTO_MS}
status=PASS
EOF

if [[ -n "$JSON_OUT" ]]; then
	cat >"$JSON_OUT" <<EOF
{
  "verify_container": "${VERIFY_CONTAINER}",
  "network_mode": "none",
  "migrations_applied": ${APPLIED_COUNT},
  "migrations_pending": ${PENDING_COUNT},
  "archiver_lag_ms": ${ARCHIVER_LAG_MS},
  "repository_archive_lag_ms": ${REPOSITORY_LAG_MS},
  "repository_newest_wal": "${REPOSITORY_WAL}",
  "local_restore_verify_ms": ${RTO_MS},
  "status": "PASS"
}
EOF
fi

if [[ "${FVPN_VERIFY_PRINT_CHECKLIST:-0}" == "1" ]]; then
	printf '%s\n' "$CHECKLIST_OUT"
fi
