#!/usr/bin/env bash
# Behavioural guard tests. All database/Docker calls below use local stubs.
# Single-quoted bash -c programs expand variables only in the tested process.
# shellcheck disable=SC2016
set -euo pipefail
scripts=$1
work=$2
requested=$3
mode=${4:-all}
mkdir -p "$work/bin"
available=$(locale -a 2>"$work/catalog-warnings")
normalized=$(printf '%s' "$requested" | tr '[:upper:]' '[:lower:]' | tr -d '-')
if ! printf '%s\n' "$available" | tr '[:upper:]' '[:lower:]' | tr -d '-' | grep -Fxq "$normalized"; then
	printf 'FAIL required locale %s is unavailable; locale -a:\n%s\n' "$requested" "$available" >&2
	exit 1
fi

# Every tested process receives explicit locale settings, including unset
# categories. Exercise LC_ALL precedence and each caller-controlled category.
test_env=(env -u LC_ALL -u LC_COLLATE -u LC_CTYPE -u LANGUAGE)
case "$mode" in
all) test_env+=(LANG=en_US.UTF-8 LC_COLLATE=en_US.UTF-8 LC_CTYPE=en_US.UTF-8 "LC_ALL=$requested") ;;
lang) test_env+=("LANG=$requested") ;;
collate) test_env+=(LANG=C.UTF-8 "LC_COLLATE=$requested" LC_CTYPE=C.UTF-8) ;;
ctype) test_env+=(LANG=C.UTF-8 LC_COLLATE=C.UTF-8 "LC_CTYPE=$requested") ;;
*) exit 2 ;;
esac
# A catalog entry or an environment string does not prove Bash loaded a locale.
# These expectations were measured with Bash on the host and in the pinned image.
case "$requested/$mode" in
C/all|C/lang) wanted='alpha=refused digit=refused length=2' ;;
C.UTF-8/all|en_US.UTF-8/ctype) wanted='alpha=refused digit=refused length=1' ;;
en_US.UTF-8/all|en_US.UTF-8/lang|en_US.UTF-8/collate) wanted='alpha=matched digit=matched length=1' ;;
*) printf 'FAIL no behaviour control for locale=%s mode=%s\n' "$requested" "$mode" >&2; exit 1 ;;
esac
control_status=0
control=$("${test_env[@]}" bash -c '
  sample=é; alpha=refused; digit=refused
  [[ "$sample" =~ ^[a-z]$ ]] && alpha=matched
  [[ ² =~ ^[0-9]$ ]] && digit=matched
  printf "alpha=%s digit=%s length=%s\n" "$alpha" "$digit" "${#sample}"
' 2>"$work/control-warnings") || control_status=$?
printf 'CONTROL locale=%s mode=%s %s\n' "$requested" "$mode" "$control"
if [[ "$control_status" != 0 || "$control" != "$wanted" || -s "$work/control-warnings" || -s "$work/catalog-warnings" ]]; then
	printf 'FAIL locale=%s mode=%s behaviour control expected %s; got %s (locale warnings forbidden)\n' "$requested" "$mode" "$wanted" "$control" >&2
	cat "$work/control-warnings" "$work/catalog-warnings" >&2
	exit 1
fi

GUARD_SNAPSHOT="$(dirname "${BASH_SOURCE[0]}")/locale-snapshot.sh"
export GUARD_SNAPSHOT
snapshot='source "$GUARD_SNAPSHOT"; snapshot_locale'
expected=$("${test_env[@]}" bash -c "$snapshot" 2>"$work/snapshot-warnings")
control_expected=$("${test_env[@]}" bash -c "LC_ALL=C; $snapshot" 2>>"$work/snapshot-warnings")
if [[ -s "$work/snapshot-warnings" ]]; then
	printf 'FAIL locale=%s mode=%s snapshot locale warnings forbidden\n' "$requested" "$mode" >&2
	cat "$work/snapshot-warnings" >&2
	exit 1
fi
export GUARD_BIN="$work/bin" GUARD_ENV_LOG="$work/commands" GUARD_CONTROL=100 GUARD_LAG=0
export PATH="$GUARD_BIN:$PATH"
export FVPN_PGBACKREST_STANZA=fvpn FVPN_PG_ARCHIVE_TIMEOUT=300s FVPN_BACKUP_DB_USER=fvpn FVPN_BACKUP_DB_NAME=fvpn
export FVPN_BACKUP_CONTAINER='' FVPN_PRIMARY_CONTAINER=''

cat >"$work/bin/stub" <<'STUB'
#!/usr/bin/env bash
set -eu
name=${0##*/}
source "$GUARD_SNAPSHOT"
printf '%s:%s\n' "$name" "$(snapshot_locale)" >>"$GUARD_ENV_LOG"
case "$name" in
pg_controldata)
  printf 'max_connections setting: %s\n' "$GUARD_CONTROL"
  printf 'max_worker_processes setting: 8\nmax_wal_senders setting: 10\nmax_prepared_xacts setting: 0\nmax_locks_per_xact setting: 64\n'
  ;;
docker)
  case "$1 $2" in
    'container inspect'|'volume inspect') exit 1 ;;
    'rm -fv'|'volume rm') exit 0 ;;
    'exec -i') printf '%s\n' "$GUARD_LAG" ;;
    *) echo GUARD_ACCEPTED >&2; exit 87 ;;
  esac ;;
pgbackrest) echo GUARD_ACCEPTED >&2; exit 87 ;;
reached) echo GUARD_ACCEPTED >&2 ;;
*) exit 99 ;;
esac
STUB
chmod 0700 "$work/bin/stub"
for name in docker pgbackrest pg_controldata reached; do
	ln -s stub "$work/bin/$name"
done

# Execute the real entrypoint's preflight unchanged, stopping before any host
# directory writes or upstream initdb. The isolated function is sourced whole.
awk '/^ISOLATED_RESTORE=/ {print "exec \"$GUARD_BIN/reached\""; found=1; exit} {print} END {if (!found) exit 1}' \
	"$scripts/postgres/fvpn-postgres-entrypoint.sh" >"$work/entry-preflight.sh"

failures=0
checks=0
check() {
	local guard=$1 value=$2 message=$3 output status=0 observed command_locale
	shift 3
	: >"$GUARD_ENV_LOG"
	output=$("${test_env[@]}" "$@" 2>&1) || status=$?
	checks=$((checks + 1))
	if [[ "$output" == *setlocale* || "$output" == *'cannot change locale'* ]]; then
		printf 'FAIL %s locale=%s mode=%s Bash locale warning: %s\n' "$guard" "$requested" "$mode" "$output"
		failures=$((failures + 1))
	fi
	if [[ -n "$message" ]]; then
		if [[ "$status" == 0 || "$output" != *"$message"* ]]; then
			printf 'FAIL %s locale=%s mode=%s value=%q expected %q; exit=%s output=%s\n' "$guard" "$requested" "$mode" "$value" "$message" "$status" "$output"
			failures=$((failures + 1))
		fi
	elif [[ "$output" != *GUARD_ACCEPTED* || ( "$status" != 0 && "$status" != 87 ) ]]; then
		printf 'FAIL %s locale=%s mode=%s ASCII value=%q did not pass; exit=%s output=%s\n' "$guard" "$requested" "$mode" "$value" "$status" "$output"
		failures=$((failures + 1))
	fi
	while IFS= read -r observed; do
		command_locale=$expected
		# This override already existed before the ASCII-guard change.
		[[ "$observed" != pg_controldata:* ]] || command_locale=$control_expected
		if [[ "${observed#*:}" != "$command_locale" ]]; then
			printf 'FAIL %s command locale changed: %s expected %s\n' "$guard" "$observed" "$command_locale"
			failures=$((failures + 1))
		fi
	done <"$GUARD_ENV_LOG"
}

run_guard() {
	local guard=$1 value=$2 rejected=$3 message
	local restore="$scripts/restore-pitr.sh" entry="$scripts/postgres/fvpn-postgres-entrypoint.sh"
	local -a args
	case "$guard" in
	restore_stanza) message='restore-pitr.sh: invalid stanza name:'; args=(bash "$restore" --latest --isolated-container valid --target-volume data --repo-volume repo --stanza "$value") ;;
	restore_time) message='restore-pitr.sh: invalid UTC timestamp'; args=(bash "$restore" --target-time "$value" --isolated-container valid --target-volume data --repo-volume repo) ;;
	restore_lsn) message='restore-pitr.sh: invalid WAL LSN'; args=(bash "$restore" --target-lsn "$value" --isolated-container valid --target-volume data --repo-volume repo) ;;
	restore_user|restore_database)
		message='restore-pitr.sh: invalid isolated --db-user or --db-name'
		local flag=--db-user
		[[ "$guard" != restore_database ]] || flag=--db-name
		args=(bash "$restore" --latest --isolated-container valid --target-volume data --repo-volume repo "$flag" "$value") ;;
	restore_container) message='restore-pitr.sh: invalid isolated container name:'; args=(bash "$restore" --latest --isolated-container "$value" --target-volume data --repo-volume repo) ;;
	entry_user|entry_database|entry_resource)
		local user=valid database=valid
		message='fvpn-postgres-entrypoint: invalid isolated database/user name'
		[[ "$guard" != entry_user ]] || user=$value
		[[ "$guard" != entry_database ]] || database=$value
		GUARD_CONTROL=100
		if [[ "$guard" == entry_resource ]]; then
			GUARD_CONTROL=$value
			message='fvpn-postgres-entrypoint: cannot determine recovery parameter max_connections from pg_controldata'
		fi
		args=(bash -c 'source "$1"; prepare_isolated_restore "$2/config" "$2/data" "$3" "$4"; "$GUARD_BIN/reached"' _ "$entry" "$work" "$user" "$database") ;;
	entry_stanza) message='fvpn-postgres-entrypoint: invalid FVPN_PGBACKREST_STANZA'; args=(env "FVPN_PGBACKREST_STANZA=$value" bash "$work/entry-preflight.sh") ;;
	entry_timeout) message='fvpn-postgres-entrypoint: invalid FVPN_PG_ARCHIVE_TIMEOUT'; args=(env "FVPN_PG_ARCHIVE_TIMEOUT=$value" bash "$work/entry-preflight.sh") ;;
	verify_prefix|verify_stanza|verify_lag)
		args=(bash "$scripts/verify.sh" --primary-container primary --repo-volume repo --migrate-bin unused --verify-prefix fvpn-w4-ops-backups-w5bic-localetest)
		case "$guard" in
		verify_prefix) message='verify.sh: invalid --verify-prefix family or run suffix'; args+=(--verify-prefix "$value") ;;
		verify_stanza) message='verify.sh: invalid stanza'; args+=(--stanza "$value") ;;
		verify_lag) message='verify.sh: archive state unknown, future-dated, or older than 15 minutes'; GUARD_LAG=$value ;;
		esac
		args=(env "FVPN_BACKUP_SCRATCH_ROOT=$work/verify" "GUARD_LAG=$GUARD_LAG" "${args[@]}") ;;
	backup_stanza) message='backup.sh: invalid stanza name:'; args=(bash "$scripts/backup.sh" --stanza "$value" info) ;;
	backup_container) message='backup.sh: invalid container name:'; args=(bash "$scripts/backup.sh" --container "$value" info) ;;
	*) exit 2 ;;
	esac
	[[ "$rejected" == 1 ]] || message=
	check "$guard" "$value" "$message" "${args[@]}"
}

# U+212A is the Kelvin sign (not ASCII K); every item is a literal test value.
non_ascii=(é ß Å ā ı İ ſ K ａ ﬁ ² ٣ ᴀ ʰ ª º Ⅰ ⓐ α а)
for guard in restore_stanza restore_time restore_lsn restore_user restore_database restore_container \
	entry_user entry_database entry_resource entry_stanza entry_timeout verify_prefix verify_stanza verify_lag backup_stanza backup_container; do
	for char in "${non_ascii[@]}"; do
		case "$guard" in
		restore_time)
			invalid=("202${char}-11-13T00:00:00Z" "2026-1${char}-13T00:00:00Z" "2026-11-1${char}T00:00:00Z"
				"2026-11-13T0${char}:00:00Z" "2026-11-13T00:0${char}:00Z" "2026-11-13T00:00:0${char}Z" "2026-11-13T00:00:00.${char}Z") ;;
		restore_lsn) invalid=("0/1${char}" "1${char}/0") ;;
		entry_resource|verify_lag) invalid=("1${char}" "$char") ;;
		entry_timeout) invalid=("1${char}s" "${char}s") ;;
		verify_prefix) invalid=("fvpn-w4-ops-backups-a${char}b" "fvpn-w4-ops-backups-w5bic-a${char}b") ;;
		*) invalid=("a${char}b" "${char}ab") ;;
		esac
		for value in "${invalid[@]}"; do run_guard "$guard" "$value" 1; done
	done
	case "$guard" in
	restore_time) valid=('2026-11-13T00:00:00.123456Z' '2026-11-13 00:00:00+00:00') ;;
	restore_lsn) valid=('01234567/89ABCDEF' 'abcdef/0') ;;
	entry_resource) valid=(0 1 2147483647) ;;
	verify_lag) valid=(0 900000) ;;
	entry_timeout) valid=(1ms 300s 3min) ;;
	verify_prefix) valid=(fvpn-w4-ops-backups-w5bic-a0123456789-b) ;;
	*user|*database) valid=(_ abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_ Z9-) ;;
	*container) valid=(abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-) ;;
	*) valid=(abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-) ;;
	esac
	for value in "${valid[@]}"; do run_guard "$guard" "$value" 0; done
	printf 'CHECKED %s locale=%s mode=%s\n' "$guard" "$requested" "$mode"
done
if [[ "$checks" != 771 ]]; then
	printf 'FAIL locale=%s mode=%s guard check count: expected 771, got %s\n' "$requested" "$mode" "$checks"
	failures=$((failures + 1))
fi
printf 'guard checks=%s failures=%s locale=%s mode=%s\n' "$checks" "$failures" "$requested" "$mode"
[[ "$failures" == 0 ]]
