// Command fvpn-release builds and signs node releases on the build machine
// (§13.3, §16.2). It must not be installed on nodes. Nodes hold only the
// public release trust store.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: fvpn-release <build-manifest|sign|verify|package|keygen>")
		return 2
	}
	var err error
	switch args[0] {
	case "build-manifest":
		err = cmdBuild(args[1:], stdout, stderr)
	case "sign":
		err = cmdSign(args[1:], stdout, stderr)
	case "verify":
		err = cmdVerify(args[1:], stdout, stderr)
	case "package":
		err = cmdPackage(args[1:], stdout, stderr)
	case "keygen":
		err = cmdKeygen(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stderr, "usage: fvpn-release <build-manifest|sign|verify|package|keygen>")
		return 0
	default:
		fmt.Fprintf(stderr, "fvpn-release: unknown command %q\n", args[0])
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "fvpn-release: %v\n", err)
		return 1
	}
	return 0
}

func cmdBuild(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build-manifest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	in := fs.String("in", "", "build input JSON")
	out := fs.String("out", "", "canonical signed object JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" || *out == "" {
		return fmt.Errorf("build-manifest requires -in and -out")
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	var input release.BuildInput
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		return fmt.Errorf("input: %w", err)
	}
	signed, manifest, err := release.BuildManifest(input, time.Now())
	if err != nil {
		return err
	}
	if err := release.WriteFile(*out, append(append([]byte{}, signed...), '\n')); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "release_id=%s\nproduct_version=%s\n", manifest.ReleaseId, manifest.ProductVersion)
	return nil
}

func cmdSign(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	fs.SetOutput(stderr)
	signedPath := fs.String("signed", "", "canonical signed object")
	keyPath := fs.String("key", "", "release private key (build machine only)")
	out := fs.String("out", "", "signed envelope JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *signedPath == "" || *keyPath == "" || *out == "" {
		return fmt.Errorf("sign requires -signed, -key and -out")
	}
	signed, err := os.ReadFile(*signedPath)
	if err != nil {
		return err
	}
	key, err := release.LoadKey(*keyPath)
	if err != nil {
		return err
	}
	_, canonical, err := release.Sign(signed, key)
	if err != nil {
		return err
	}
	if err := release.WriteFile(*out, append(append([]byte{}, canonical...), '\n')); err != nil {
		return err
	}
	sum, err := release.EnvelopeSHA256(canonical)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "key_id=%s\nmanifest_sha256=%s\n", key.KeyID, sum)
	return nil
}

func cmdVerify(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envPath := fs.String("envelope", "", "signed envelope")
	trustPath := fs.String("trust", "", "public release trust store")
	at := fs.String("at", "", "verification time, RFC3339 (default: now)")
	var artifacts artifactFlags
	fs.Var(&artifacts, "artifact", "arch=path check, repeatable (amd64, arm64)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *envPath == "" || *trustPath == "" {
		return fmt.Errorf("verify requires -envelope and -trust")
	}
	now := time.Now().UTC()
	if *at != "" {
		t, err := time.Parse(time.RFC3339, *at)
		if err != nil {
			return fmt.Errorf("-at: %w", err)
		}
		now = t
	}
	raw, err := os.ReadFile(*envPath)
	if err != nil {
		return err
	}
	trustRaw, err := os.ReadFile(*trustPath)
	if err != nil {
		return err
	}
	ts, err := envelope.ParseTrustStoreJSON(trustRaw)
	if err != nil {
		return err
	}
	signed, keyID, err := release.Verify(raw, ts, now)
	if err != nil {
		return err
	}
	manifest, err := release.ParsedManifest(signed)
	if err != nil {
		return err
	}
	for _, a := range artifacts {
		if err := release.CheckArtifact(manifest, a.arch, a.path); err != nil {
			return err
		}
	}
	sum, err := release.EnvelopeSHA256(raw)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "release_id=%s\nkey_id=%s\nmanifest_sha256=%s\n", manifest.ReleaseId, keyID, sum)
	return nil
}

func cmdPackage(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "", "release tree")
	out := fs.String("out", "", "tar.zst output")
	mtime := fs.Int64("mtime", 0, "unix mtime applied to every entry")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" || *out == "" {
		return fmt.Errorf("package requires -dir and -out")
	}
	when := time.Unix(*mtime, 0).UTC()
	if err := release.PackageDir(*dir, *out, when); err != nil {
		return err
	}
	sum, size, err := hashOut(*out)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "sha256=%s\nsize_bytes=%d\n", sum, size)
	return nil
}

func cmdKeygen(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	keyID := fs.String("key-id", "", "public key id")
	out := fs.String("out", "", "private key path (build machine only)")
	trustOut := fs.String("trust-out", "", "public trust store path, optional")
	nb := fs.String("not-before", "", "trust window start, RFC3339")
	na := fs.String("not-after", "", "trust window end, RFC3339")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *keyID == "" || *out == "" {
		return fmt.Errorf("keygen requires -key-id and -out")
	}
	key, err := release.GenerateKey(*keyID)
	if err != nil {
		return err
	}
	if err := key.Save(*out); err != nil {
		return err
	}
	if *trustOut != "" {
		if *nb == "" || *na == "" {
			return fmt.Errorf("keygen -trust-out requires -not-before and -not-after")
		}
		notBefore, err := time.Parse(time.RFC3339, *nb)
		if err != nil {
			return err
		}
		notAfter, err := time.Parse(time.RFC3339, *na)
		if err != nil {
			return err
		}
		ts, err := key.TrustStore(notBefore, notAfter)
		if err != nil {
			return err
		}
		raw, err := json.MarshalIndent(ts, "", "  ")
		if err != nil {
			return err
		}
		if err := release.WriteFile(*trustOut, append(raw, '\n')); err != nil {
			return err
		}
	}
	fmt.Fprintf(stdout, "key_id=%s\npurpose=release\n", key.KeyID)
	fmt.Fprintln(stderr, "fvpn-release: private key stays on the build machine; do not copy it to nodes")
	return nil
}

type artifactFlag struct {
	arch string
	path string
}

type artifactFlags []artifactFlag

func (a *artifactFlags) String() string { return "" }

func (a *artifactFlags) Set(v string) error {
	arch, path, ok := strings.Cut(v, "=")
	if !ok || arch == "" || path == "" {
		return fmt.Errorf("-artifact must be arch=path")
	}
	*a = append(*a, artifactFlag{arch: arch, path: path})
	return nil
}

func hashOut(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", 0, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), st.Size(), nil
}
