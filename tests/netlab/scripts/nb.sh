#!/bin/bash
# Simulated NetBird rules inside fv-fwd. This is not table inet fvpn.
# The outer enforcement on the host must still hold (§8.2, §10.2).
set -euo pipefail

mode="${1:-on}"
ns() { ip netns exec fv-fwd "$@"; }

case "$mode" in
on)
	ns nft delete table inet fvnb >/dev/null 2>&1 || true
	ns nft -f - <<'EOF'
table inet fvnb {
	chain forward {
		type filter hook forward priority 0; policy accept;
		accept
	}
	chain postrouting {
		type nat hook postrouting priority 100; policy accept;
	}
}
EOF
	;;
masq)
	ns nft delete table inet fvnb >/dev/null 2>&1 || true
	ns nft -f - <<'EOF'
table inet fvnb {
	chain forward {
		type filter hook forward priority 0; policy accept;
		accept
	}
	chain postrouting {
		type nat hook postrouting priority 100; policy accept;
		oifname "fv-ff" masquerade
	}
}
EOF
	;;
off)
	ns nft delete table inet fvnb >/dev/null 2>&1 || true
	;;
*)
	echo "usage: nb.sh on|masq|off" >&2
	exit 1
	;;
esac
