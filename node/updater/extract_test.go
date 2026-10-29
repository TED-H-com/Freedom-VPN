package updater

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/klauspost/compress/zstd"
)

type tarMember struct {
	name string
	body string
	typ  byte
	link string
	mode int64
}

func packTar(t *testing.T, members []tarMember) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(enc)
	for _, m := range members {
		mode := m.mode
		if mode == 0 {
			mode = 0o644
		}
		hdr := &tar.Header{Name: m.name, Mode: mode, Typeflag: m.typ}
		if m.typ == 0 {
			hdr.Typeflag = tar.TypeReg
		}
		if hdr.Typeflag == tar.TypeReg {
			hdr.Size = int64(len(m.body))
		}
		hdr.Linkname = m.link
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(m.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func TestUPD_ExtractRejectsHostilePaths(t *testing.T) {
	dest := t.TempDir()
	outside := filepath.Join(filepath.Dir(dest), "outside-secret")
	cases := []struct {
		name string
		mem  tarMember
	}{
		{"absolute", tarMember{name: "/tmp/evil", body: "x"}},
		{"dotdot", tarMember{name: "../outside-secret", body: "x"}},
		{"nested dotdot", tarMember{name: "foo/../../outside-secret", body: "x"}},
		{"symlink escape", tarMember{name: "escape", typ: tar.TypeSymlink, link: "../outside-secret"}},
		{"absolute symlink", tarMember{name: "abs", typ: tar.TypeSymlink, link: "/tmp/evil"}},
		{"device", tarMember{name: "dev", typ: tar.TypeChar, mode: 0o644}},
		{"setuid", tarMember{name: "su", body: "x", mode: 0o4755}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sub := filepath.Join(dest, tc.name)
			err := ExtractArchive(packTar(t, []tarMember{tc.mem}), sub, ExtractOptions{MaxBytes: 1 << 20})
			if err == nil {
				t.Fatal("extract succeeded")
			}
			if _, statErr := os.Stat(outside); !os.IsNotExist(statErr) {
				t.Fatalf("hostile member escaped: %v", statErr)
			}
		})
	}
}

func TestUPD_ExtractRejectsRelativeSymlink(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "rel")
	err := ExtractArchive(packTar(t, []tarMember{
		{name: "bin/tool", body: "tool", mode: 0o755},
		{name: "bin/link", typ: tar.TypeSymlink, link: "tool"},
	}), dest, ExtractOptions{MaxBytes: 1 << 20})
	if err == nil {
		t.Fatal("relative symlink was extracted")
	}
	if _, statErr := os.Lstat(filepath.Join(dest, "bin/link")); !os.IsNotExist(statErr) {
		t.Fatalf("symlink left behind: %v", statErr)
	}
}

func TestUPD_ExtractExactInstalledSize(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(filepath.Join(tree, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	agent := bytes.Repeat([]byte("A"), 4000)
	updater := bytes.Repeat([]byte("U"), 4000)
	readme := []byte("Freedom VPN release notes\n")
	if err := os.WriteFile(filepath.Join(tree, "bin", "fvpn-agent"), agent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "bin", "fvpn-updater"), updater, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "README"), readme, 0o644); err != nil {
		t.Fatal(err)
	}
	installed := int64(len(agent) + len(updater) + len(readme))
	arc := filepath.Join(root, "release.tar.zst")
	if err := release.PackageDir(tree, arc, time.Unix(1_700_000_000, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(arc)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "out")
	if err := ExtractArchive(bytes.NewReader(raw), dest, ExtractOptions{MaxBytes: installed}); err != nil {
		t.Fatal(err)
	}
	if got := fileBytes(t, filepath.Join(dest, "bin", "fvpn-agent")); !bytes.Equal(got, agent) {
		t.Fatalf("agent bytes %d", len(got))
	}
}

func TestUPD_ExtractAcceptsDotRoot(t *testing.T) {
	raw := packTar(t, []tarMember{
		{name: "./", typ: tar.TypeDir, mode: 0o755},
		{name: "./bin/", typ: tar.TypeDir, mode: 0o755},
		{name: "./bin/fvpn-agent", body: "agent", mode: 0o755},
	})
	dest := filepath.Join(t.TempDir(), "out")
	if err := ExtractArchive(raw, dest, ExtractOptions{MaxBytes: 1 << 20}); err != nil {
		t.Fatal(err)
	}
	if got := string(fileBytes(t, filepath.Join(dest, "bin", "fvpn-agent"))); got != "agent" {
		t.Fatalf("body %q", got)
	}
}

func TestUPD_ExtractRejectsHardLink(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out")
	err := ExtractArchive(packTar(t, []tarMember{
		{name: "bin/tool", body: "tool", mode: 0o755},
		{name: "bin/alias", typ: tar.TypeLink, link: "bin/tool"},
	}), dest, ExtractOptions{MaxBytes: 1 << 20})
	if err == nil {
		t.Fatal("hard link was extracted")
	}
}
