#!/bin/sh
# Freedom VPN node bootstrap (§6.3, §18.2, PKG01).
#
# Downloads one named release archive and its signed manifest envelope,
# checks the embedded installer pin, and runs fvpn-node-install verify-archive
# from that archive. Signature checks stay inside the binary, which uses
# packaging/release. This script does not install packages, enroll a node,
# read a token, or keep a secret.
#
# The release pipeline replaces @@INSTALLER_SHA256@@ and the empty trust
# store between the markers. A placeholder or an empty store is refused.
#
# Usage:
#   install.sh --manifest-url URL --artifact-url URL --out DIR -- <installer args>
#
# URLs must be https, or http to 127.0.0.1 / localhost for a lab harness.
# The last path segment must not be "latest". --out is the directory the
# operator names in the installer arguments; this script does not invent
# install flags.
set -eu

installer_sha256='@@INSTALLER_SHA256@@'

die() {
    printf '%s\n' "$1" >&2
    exit 64
}

extract_trust() {
    awk '
        $0 == "# BEGIN TRUST ROOT" {capture = 1; next}
        $0 == "# END TRUST ROOT" {capture = 0; next}
        capture {print}
    ' "$0"
}

case "$installer_sha256" in
*@@*|''|*[!0-9a-fA-F]*)
    die "bootstrap: installer pin is missing"
    ;;
esac
if [ "${#installer_sha256}" -ne 64 ]; then
    die "bootstrap: installer pin is missing"
fi

trust_json=$(extract_trust)
trust_flat=$(printf '%s' "$trust_json" | tr -d '[:space:]')
case "$trust_flat" in
''|*@@TRUST_ROOT@@*|*'{"keys":[]}'*)
    die "bootstrap: trust root is missing"
    ;;
esac

manifest_url=''
artifact_url=''
out_dir=''
while [ "$#" -gt 0 ]; do
    case "$1" in
    --manifest-url)
        shift
        manifest_url=${1:-}
        ;;
    --artifact-url)
        shift
        artifact_url=${1:-}
        ;;
    --out)
        shift
        out_dir=${1:-}
        ;;
    --)
        shift
        break
        ;;
    *)
        die "bootstrap: unknown argument"
        ;;
    esac
    shift
done

[ -n "$manifest_url" ] && [ -n "$artifact_url" ] && [ -n "$out_dir" ] || die "bootstrap: manifest URL, artifact URL and --out are required"

refuse_url() {
    url=$1
    case "$url" in
    *[[:space:]]*|*'?'*|*'#'*)
        die "bootstrap: refusing URL"
        ;;
    https://*)
        ;;
    http://127.0.0.1/*|http://127.0.0.1:*)
        ;;
    http://localhost/*|http://localhost:*)
        ;;
    *)
        die "bootstrap: refusing URL"
        ;;
    esac
    base=${url%/}
    base=${base##*/}
    case "$base" in
    latest|latest.*)
        die "bootstrap: refusing latest"
        ;;
    esac
}

refuse_url "$manifest_url"
refuse_url "$artifact_url"

for tool in curl zstd tar sha256sum; do
    command -v "$tool" >/dev/null 2>&1 || die "bootstrap: missing $tool"
done

mkdir -p "$out_dir"
chmod 0700 "$out_dir"
manifest="$out_dir/manifest.json"
artifact="$out_dir/artifact.tar.zst"
installer="$out_dir/fvpn-node-install"
trust="$out_dir/trust.json"

# No redirect follow: a redirect onto a latest name would skip refuse_url.
curl -fsS --proto '=https,http' --max-time 120 -o "$manifest" "$manifest_url"
curl -fsS --proto '=https,http' --max-time 300 -o "$artifact" "$artifact_url"
printf '%s\n' "$trust_json" >"$trust"
chmod 0600 "$trust" "$manifest"

rm -rf "$out_dir/extract"
mkdir -p "$out_dir/extract"
zstd -d -c "$artifact" | tar -x -f - -C "$out_dir/extract" fvpn-node-install
got=$(sha256sum "$out_dir/extract/fvpn-node-install" | awk 'NR==1 {print $1}')
if [ "$got" != "$installer_sha256" ]; then
    die "bootstrap: installer pin does not match"
fi
mv "$out_dir/extract/fvpn-node-install" "$installer"
rmdir "$out_dir/extract"
chmod 0755 "$installer"

"$installer" verify-archive --manifest "$manifest" --artifact "$artifact" --trust "$trust"
exec "$installer" "$@"
exit 1

# BEGIN TRUST ROOT
@@TRUST_ROOT@@
# END TRUST ROOT
