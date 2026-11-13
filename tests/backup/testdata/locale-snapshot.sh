#!/usr/bin/env bash
# Serialize all locale variables, including empty versus unset and future LC_*.
snapshot_locale() {
	local variable
	while IFS= read -r variable; do
		if [[ -n "$variable" && -v "$variable" ]]; then
			printf '%s=%q;' "$variable" "${!variable}"
		fi
	done < <(printf '%s\n' LANG LANGUAGE "${!LC_@}" | LC_ALL=C sort -u)
	printf '\n'
}
