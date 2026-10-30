#!/usr/bin/env bash
# Freedom VPN Point-in-Time Restore (PITR) Tool (`deploy/backup/restore-pitr.sh`)
# Spec references: §15.4, §17.2, §17.3, §17.4, AT30 (SECOPS-05)
#
# Restores a physical pgBackRest backup + WAL stream to a specific UTC timestamp
# or LSN into either:
#   1. An isolated Docker container (`--network none`, zero published host ports), or
#   2. A dedicated local target data directory (`--target-dir`).
#
# Safety contract:
#   NEVER restores over a live primary instance or an active data directory.
set -euo pipefail

STANZA="${FVPN_PGBACKREST_STANZA:-fvpn}"
TARGET_TIME=""
TARGET_LSN=""
RESTORE_LATEST=0
TARGET_EXCLUSIVE=0
ISOLATED_CONTAINER=""
TARGET_VOLUME=""
REPO_VOLUME=""
PRIMARY_CONTAINER="${FVPN_PRIMARY_CONTAINER:-}"
IMAGE="${FVPN_BACKUP_IMAGE:-fvpn-w4-ops-backups-postgres:17}"
TARGET_DIR=""
PRIMARY_DIR="${FVPN_PRIMARY_PGDATA:-}"
REPO_PATH=""
DB_USER="${FVPN_BACKUP_DB_USER:-fvpn}"
DB_NAME="${FVPN_BACKUP_DB_NAME:-fvpn}"
CIPHER_PASS="${FVPN_PGBACKREST_CIPHER_PASS:-}"

usage() {
	cat >&2 <<'EOF'
Usage:
  restore-pitr.sh [target-option] [destination-option] [options]

Target options (choose one):
  --target-time <timestamp>       UTC recovery timestamp (e.g. 2026-10-29T18:02:00.123456Z)
  --target-lsn <lsn>              PostgreSQL WAL LSN (e.g. 0/204D2E8)
  --latest                        Restore latest backup and replay all available WAL

Destination options (choose one):
  --isolated-container <name>     Start an isolated container (--network none, no host ports)
    --target-volume <volume>      Fresh Docker volume for restored PGDATA
    --repo-volume <vol_or_path>   pgBackRest repository volume or directory (mounted :ro)
  --target-dir <path>             Restore into an empty local directory (never a live PGDATA)

Additional options:
  --stanza <name>                 pgBackRest stanza name (default: fvpn)
  --image <image>                 PostgreSQL + pgBackRest image for isolated restore
  --primary-container <name>      Live primary container name (explicit safety guard)
  --primary-dir <path>            Live primary PGDATA path (explicit safety guard)
  --repo-path <path>              Host pgBackRest repo path for --target-dir mode
  --target-exclusive              Stop immediately before the target time/LSN
  --db-user <user>                PostgreSQL user (default: fvpn)
  --db-name <db>                  PostgreSQL database (default: fvpn)
  -h, --help                      Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--stanza)
		STANZA="${2:-}"
		shift 2
		;;
	--target-time)
		TARGET_TIME="${2:-}"
		shift 2
		;;
	--target-lsn)
		TARGET_LSN="${2:-}"
		shift 2
		;;
	--latest)
		RESTORE_LATEST=1
		shift
		;;
	--target-exclusive)
		TARGET_EXCLUSIVE=1
		shift
		;;
	--isolated-container)
		ISOLATED_CONTAINER="${2:-}"
		shift 2
		;;
	--target-volume)
		TARGET_VOLUME="${2:-}"
		shift 2
		;;
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
	--target-dir)
		TARGET_DIR="${2:-}"
		shift 2
		;;
	--primary-dir)
		PRIMARY_DIR="${2:-}"
		shift 2
		;;
	--repo-path)
		REPO_PATH="${2:-}"
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
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "restore-pitr.sh: unknown argument: $1" >&2
		usage
		exit 2
		;;
	esac
done

if [[ ! "$STANZA" =~ ^[a-zA-Z0-9_-]+$ ]]; then
	echo "restore-pitr.sh: invalid stanza name: $STANZA" >&2
	exit 2
fi

# Validate that exactly one recovery target mode was chosen.
TARGET_MODES=0
[[ -n "$TARGET_TIME" ]] && TARGET_MODES=$((TARGET_MODES + 1))
[[ -n "$TARGET_LSN" ]] && TARGET_MODES=$((TARGET_MODES + 1))
[[ "$RESTORE_LATEST" -eq 1 ]] && TARGET_MODES=$((TARGET_MODES + 1))
if [[ "$TARGET_MODES" -ne 1 ]]; then
	echo "restore-pitr.sh: specify exactly one of --target-time, --target-lsn, or --latest" >&2
	exit 2
fi

PGBACKREST_TARGET_ARGS=()
if [[ -n "$TARGET_TIME" ]]; then
	# Accept strict UTC RFC 3339 (YYYY-MM-DDTHH:MM:SS[.ffffff]Z) or PG UTC (YYYY-MM-DD HH:MM:SS[.ffffff]+00).
	if [[ "$TARGET_TIME" =~ ^([0-9]{4}-[0-9]{2}-[0-9]{2})[T\ ]([0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?)(Z|\+00(:00)?)$ ]]; then
		PG_TIME="${BASH_REMATCH[1]} ${BASH_REMATCH[2]}+00"
		PGBACKREST_TARGET_ARGS=(--type=time "--target=${PG_TIME}" --target-action=promote)
	else
		echo "restore-pitr.sh: invalid UTC timestamp '${TARGET_TIME}' (expected YYYY-MM-DDTHH:MM:SS[.ffffff]Z)" >&2
		exit 2
	fi
elif [[ -n "$TARGET_LSN" ]]; then
	if [[ "$TARGET_LSN" =~ ^[0-9A-Fa-f]{1,8}/[0-9A-Fa-f]{1,8}$ ]]; then
		PGBACKREST_TARGET_ARGS=(--type=lsn "--target=${TARGET_LSN}" --target-action=promote)
	else
		echo "restore-pitr.sh: invalid WAL LSN '${TARGET_LSN}' (expected X/YYYYYYYY)" >&2
		exit 2
	fi
else
	PGBACKREST_TARGET_ARGS=(--type=default)
fi

if [[ "$TARGET_EXCLUSIVE" -eq 1 ]]; then
	PGBACKREST_TARGET_ARGS+=(--target-exclusive)
fi

# Validate destination mode (isolated container vs target directory).
if [[ -n "$ISOLATED_CONTAINER" ]] && [[ -n "$TARGET_DIR" ]]; then
	echo "restore-pitr.sh: --isolated-container and --target-dir are mutually exclusive" >&2
	exit 2
fi
if [[ -z "$ISOLATED_CONTAINER" ]] && [[ -z "$TARGET_DIR" ]]; then
	echo "restore-pitr.sh: specify either --isolated-container or --target-dir" >&2
	exit 2
fi

if [[ -n "$TARGET_DIR" ]]; then
	# Host directory mode: enforce live-primary refusal guards.
	RESOLVED_TARGET="$(realpath -m "$TARGET_DIR")"
	if [[ -n "$PRIMARY_DIR" ]]; then
		RESOLVED_PRIMARY="$(realpath -m "$PRIMARY_DIR")"
		if [[ "$RESOLVED_TARGET" == "$RESOLVED_PRIMARY" ]]; then
			echo "restore-pitr.sh: refusing to restore over live primary directory '${TARGET_DIR}'" >&2
			exit 2
		fi
	fi
	if [[ -n "${PGDATA:-}" ]]; then
		RESOLVED_PGDATA="$(realpath -m "$PGDATA")"
		if [[ "$RESOLVED_TARGET" == "$RESOLVED_PGDATA" ]]; then
			echo "restore-pitr.sh: refusing to restore over active PGDATA '${PGDATA}'" >&2
			exit 2
		fi
	fi
	if [[ -f "${TARGET_DIR}/postmaster.pid" ]]; then
		echo "restore-pitr.sh: refusing to restore over a live PostgreSQL instance (postmaster.pid found in '${TARGET_DIR}')" >&2
		exit 2
	fi
	if [[ -d "$TARGET_DIR" ]] && [[ -n "$(ls -A "$TARGET_DIR" 2>/dev/null)" ]]; then
		echo "restore-pitr.sh: target directory '${TARGET_DIR}' is not empty; refusing to overwrite" >&2
		exit 2
	fi
	mkdir -p "$TARGET_DIR"
	chmod 0700 "$TARGET_DIR"

	EXTRA_ARGS=(--pg1-path="$TARGET_DIR")
	if [[ -n "$REPO_PATH" ]]; then
		EXTRA_ARGS+=(--repo1-path="$REPO_PATH")
	fi
	pgbackrest --stanza="$STANZA" "${EXTRA_ARGS[@]}" "${PGBACKREST_TARGET_ARGS[@]}" restore
	exit 0
fi

# Isolated container mode: validate identifiers and enforce live-primary refusal guards.
if [[ ! "$ISOLATED_CONTAINER" =~ ^[a-zA-Z0-9_.-]+$ ]]; then
	echo "restore-pitr.sh: invalid isolated container name: $ISOLATED_CONTAINER" >&2
	exit 2
fi
if [[ -z "$TARGET_VOLUME" ]] || [[ -z "$REPO_VOLUME" ]]; then
	echo "restore-pitr.sh: --isolated-container requires both --target-volume and --repo-volume" >&2
	exit 2
fi
if [[ -n "$PRIMARY_CONTAINER" ]] && [[ "$ISOLATED_CONTAINER" == "$PRIMARY_CONTAINER" ]]; then
	echo "restore-pitr.sh: refusing to restore over live primary container '${PRIMARY_CONTAINER}'" >&2
	exit 2
fi

EXISTING_CONTAINER="$(docker ps -a --filter "name=^/${ISOLATED_CONTAINER}$" --format '{{.Names}}')"
if [[ -n "$EXISTING_CONTAINER" ]]; then
	echo "restore-pitr.sh: container '${ISOLATED_CONTAINER}' already exists; refusing to overwrite live or existing container" >&2
	exit 2
fi

VOLUME_IN_USE="$(docker ps -a --filter "volume=${TARGET_VOLUME}" --format '{{.Names}}')"
if [[ -n "$VOLUME_IN_USE" ]]; then
	echo "restore-pitr.sh: target volume '${TARGET_VOLUME}' is mounted by container '${VOLUME_IN_USE}'; refusing to overwrite" >&2
	exit 2
fi

docker volume create "$TARGET_VOLUME" >/dev/null

# Prepare cipher flags if repository encryption is configured.
CIPHER_ARGS=()
if [[ -n "$CIPHER_PASS" ]]; then
	CIPHER_ARGS=(--repo1-cipher-type=aes-256-cbc "--repo1-cipher-pass=${CIPHER_PASS}")
fi

# Step 1: Run pgbackrest restore in a one-shot network-none container with the backup repo mounted read-only (:ro).
RESTORE_HELPER="${ISOLATED_CONTAINER}-init"
docker rm -f "$RESTORE_HELPER" >/dev/null 2>&1 || true
docker run --rm \
	--name "$RESTORE_HELPER" \
	--network none \
	--entrypoint /bin/bash \
	-v "${TARGET_VOLUME}:/var/lib/postgresql/data" \
	-v "${REPO_VOLUME}:/var/lib/pgbackrest:ro" \
	"$IMAGE" \
	-c '
		set -euo pipefail
		stanza="$1"
		shift
		chown postgres:postgres /var/lib/postgresql/data
		chmod 0700 /var/lib/postgresql/data
		find /var/lib/postgresql/data -mindepth 1 -delete
		exec runuser -u postgres -- pgbackrest \
			--stanza="$stanza" \
			--pg1-path=/var/lib/postgresql/data \
			--repo1-path=/var/lib/pgbackrest \
			--log-path=/tmp \
			"$@" \
			restore
	' _ "$STANZA" "${CIPHER_ARGS[@]}" "${PGBACKREST_TARGET_ARGS[@]}"

# Step 2: Start the isolated PostgreSQL container with --network none and zero host ports.
# PostgreSQL replays archived WAL from /var/lib/pgbackrest:ro up to the PITR target and promotes.
docker run -d \
	--name "$ISOLATED_CONTAINER" \
	--network none \
	-e FVPN_PG_ISOLATED_RESTORE=1 \
	-e FVPN_PGBACKREST_STANZA="$STANZA" \
	-e FVPN_PGBACKREST_CIPHER_PASS="$CIPHER_PASS" \
	-e POSTGRES_USER="$DB_USER" \
	-e POSTGRES_DB="$DB_NAME" \
	-e POSTGRES_HOST_AUTH_METHOD=trust \
	-v "${TARGET_VOLUME}:/var/lib/postgresql/data" \
	-v "${REPO_VOLUME}:/var/lib/pgbackrest:ro" \
	"$IMAGE" >/dev/null

# Verify isolation invariants: NetworkMode must be "none" and zero host ports may be published.
NET_MODE="$(docker inspect --format '{{.HostConfig.NetworkMode}}' "$ISOLATED_CONTAINER")"
PORT_COUNT="$(docker inspect --format '{{len .NetworkSettings.Ports}}' "$ISOLATED_CONTAINER")"
if [[ "$NET_MODE" != "none" ]] || [[ "$PORT_COUNT" != "0" ]]; then
	docker rm -fv "$ISOLATED_CONTAINER" >/dev/null 2>&1 || true
	echo "restore-pitr.sh: isolation invariant violated (NetworkMode=${NET_MODE}, Ports=${PORT_COUNT})" >&2
	exit 1
fi

# Step 3: Wait for PostgreSQL to complete WAL replay and promote out of recovery.
PROMOTED=0
for _ in $(seq 1 120); do
	STATE="$(docker inspect --format '{{.State.Running}}' "$ISOLATED_CONTAINER" 2>/dev/null || echo "false")"
	if [[ "$STATE" != "true" ]]; then
		echo "restore-pitr.sh: isolated container '${ISOLATED_CONTAINER}' exited unexpectedly during recovery:" >&2
		docker logs "$ISOLATED_CONTAINER" >&2 || true
		exit 1
	fi
	if READY="$(docker exec --user postgres "$ISOLATED_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT CASE WHEN NOT pg_is_in_recovery() THEN 'promoted' ELSE 'recovering' END;" 2>/dev/null | tr -d '[:space:]')" && [[ "$READY" == "promoted" ]]; then
		PROMOTED=1
		break
	fi
	sleep 0.25
done

if [[ "$PROMOTED" -ne 1 ]]; then
	echo "restore-pitr.sh: timed out waiting for isolated container '${ISOLATED_CONTAINER}' to finish PITR recovery and promote" >&2
	docker logs "$ISOLATED_CONTAINER" >&2 || true
	exit 1
fi
