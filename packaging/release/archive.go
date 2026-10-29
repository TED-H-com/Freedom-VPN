package release

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

// zstdOptions is the fixed encoder configuration. Concurrency 1 keeps the
// frame deterministic; the window and checksum do not depend on the clock
// or on the host CPU count (§13.3 immutable bytes).
func zstdOptions() []zstd.EOption {
	return []zstd.EOption{
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderConcurrency(1),
		zstd.WithWindowSize(1 << 20),
		zstd.WithEncoderCRC(true),
		zstd.WithZeroFrames(false),
	}
}

// PackageDir writes a reproducible tar.zst of dir to out. Entry order,
// owners and mtimes are fixed: paths sort lexicographically, uid and gid are
// 0, uname and gname are root, and every mtime is the given UTC time.
// Symlinks, devices and irregular files are refused rather than archived.
func PackageDir(dir, out string, mtime time.Time) error {
	if mtime.IsZero() {
		mtime = time.Unix(0, 0).UTC()
	}
	mtime = mtime.UTC()
	entries, err := collect(dir)
	if err != nil {
		return err
	}
	tmp := out + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()
	enc, err := zstd.NewWriter(f, zstdOptions()...)
	if err != nil {
		return err
	}
	tw := tar.NewWriter(enc)
	for _, e := range entries {
		if err := writeEntry(tw, dir, e, mtime); err != nil {
			_ = tw.Close()
			_ = enc.Close()
			return err
		}
	}
	if err := tw.Close(); err != nil {
		_ = enc.Close()
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, out); err != nil {
		return err
	}
	ok = true
	return nil
}

type entry struct {
	rel  string
	info os.FileInfo
}

func collect(dir string) ([]entry, error) {
	var out []entry
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			return fmt.Errorf("release: %s is a symlink; release trees contain regular files and directories only", rel)
		case mode&os.ModeDevice != 0 || mode&os.ModeNamedPipe != 0 || mode&os.ModeSocket != 0 || mode&os.ModeIrregular != 0:
			return fmt.Errorf("release: %s is not a regular file or directory", rel)
		case !mode.IsDir() && !mode.IsRegular():
			return fmt.Errorf("release: %s is not a regular file or directory", rel)
		}
		out = append(out, entry{rel: filepath.ToSlash(rel), info: info})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out, nil
}

func writeEntry(tw *tar.Writer, root string, e entry, mtime time.Time) error {
	hdr := &tar.Header{
		Name:    e.rel,
		ModTime: mtime,
		Format:  tar.FormatPAX,
		Uid:     0,
		Gid:     0,
		Uname:   "root",
		Gname:   "root",
	}
	if e.info.IsDir() {
		if !strings.HasSuffix(hdr.Name, "/") {
			hdr.Name += "/"
		}
		hdr.Typeflag = tar.TypeDir
		hdr.Mode = 0o755
		return tw.WriteHeader(hdr)
	}
	mode := int64(0o644)
	if e.info.Mode()&0o111 != 0 {
		mode = 0o755
	}
	hdr.Typeflag = tar.TypeReg
	hdr.Mode = mode
	hdr.Size = e.info.Size()
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(e.rel)))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

// ReadArchive lists a tar.zst for tests and for verify-time inspection.
// It does not extract and does not follow links.
func ReadArchive(path string) ([]tar.Header, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec, err := zstd.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	tr := tar.NewReader(dec)
	var out []tar.Header
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, *hdr)
		if _, err := io.Copy(io.Discard, tr); err != nil {
			return nil, err
		}
	}
}
