// Package release builds, signs, verifies and packages Freedom VPN node
// releases (§13.3, §16.2). It runs on the build machine. Nodes never receive
// the release signing private key; they only verify with pkg/envelope.
//
// Artifact format is tar.zst because that is the only value
// ManifestArtifact.format allows. Compression uses klauspost/compress, a
// pure-Go zstd implementation, with a fixed encoder configuration so two
// builds of the same tree produce the same bytes.
package release
