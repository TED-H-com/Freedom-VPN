// Package release builds, signs, verifies and packages Freedom VPN node
// releases (§13.3, §16.2). It runs on the build machine. Nodes never receive
// the release signing private key; they only verify with pkg/envelope.
//
// The fvpn-release keygen command validates and serializes requested public
// trust before writing either output. Invalid trust input leaves existing
// outputs unchanged. Output files may still be replaced on a valid invocation,
// and a filesystem error writing public trust can leave a saved private key;
// inspect the outputs before retrying after an I/O failure.
//
// Artifact format is tar.zst because that is the only value
// ManifestArtifact.format allows. Compression uses klauspost/compress, a
// pure-Go zstd implementation, with a fixed encoder configuration so two
// builds of the same tree produce the same bytes.
package release
