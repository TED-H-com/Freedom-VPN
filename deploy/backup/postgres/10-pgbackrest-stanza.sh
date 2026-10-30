#!/usr/bin/env bash
# Initializes the local pgBackRest stanza during initial PostgreSQL cluster bootstrap
# so archive_command succeeds immediately when the primary starts (§15.4, §17.2).
set -euo pipefail

if [[ "${FVPN_PG_ISOLATED_RESTORE:-0}" == "1" ]] || [[ "${FVPN_PGBACKREST_AUTO_STANZA:-1}" == "0" ]]; then
	exit 0
fi

STANZA="${FVPN_PGBACKREST_STANZA:-fvpn}"
if [[ -w /var/lib/pgbackrest ]]; then
	pgbackrest --stanza="${STANZA}" stanza-create
fi
