// Package installer is the Ubuntu node installer (fvpn-node-install).
//
// It implements the installer contract in §6.3 and PKG01: identify the
// platform, check tools, install a verified release, enroll through
// fvpn-agent, and only then start services. It is idempotent (§6.3, AT07),
// journals every host change under /var/lib/vpn-node/install, and rolls
// back only what it changed. It never flushes the host nftables ruleset
// and never rewrites the host default route or the system resolver.
//
// Signature checks call packaging/release. The exclusive lock is the same
// file and flock protocol as node/internal/nodelock (§13.5). This package
// cannot import that internal package, so lock.go repeats the protocol.
// The updater has no local-apply command; upgrade stages a verified tree
// and starts fvpn-updater run.
package installer
