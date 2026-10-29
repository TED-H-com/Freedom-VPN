package updater

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

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

func TestUPD_ExtractAllowsRelativeSymlink(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "rel")
	err := ExtractArchive(packTar(t, []tarMember{
		{name: "bin/tool", body: "tool", mode: 0o755},
		{name: "bin/link", typ: tar.TypeSymlink, link: "tool"},
	}), dest, ExtractOptions{MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(filepath.Join(dest, "bin/link"))
	if err != nil || got != "tool" {
		t.Fatalf("link %q err %v", got, err)
	}
}
