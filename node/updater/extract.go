package updater

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/klauspost/compress/zstd"
)

// maxArchiveEntries is the default member cap (D-034a, D-034f).
const maxArchiveEntries = 4096

// defaultDecompressedCap bounds the extracted file bytes when MaxBytes is unset.
const defaultDecompressedCap = 512 << 20

// archiveEntryOverhead is the per-member allowance on the decompressed tar
// stream: a PAX header, the 512-byte ustar header and block padding, with margin.
const archiveEntryOverhead int64 = 8 << 10

// archiveTrailer is the two 512-byte zero blocks that end a tar, with margin.
const archiveTrailer int64 = 2 << 10

// ExtractOptions bounds a safe extract (§13.5). MaxBytes caps the sum of
// extracted regular-file bytes, which is the manifest's installed_size_bytes.
// The decompressed tar stream is bounded separately at
// MaxBytes + MaxEntries*archiveEntryOverhead + archiveTrailer, so headers and
// padding do not count against the installed size. MaxEntries caps archive
// members; zero means maxArchiveEntries. Deny lists persistent directories
// the archive must not touch.
type ExtractOptions struct {
	MaxBytes   int64
	MaxEntries int
	Deny       []string
	// DiskFull forces ENOSPC-equivalent failure before any file is written.
	// Tests use it; production leaves it false.
	DiskFull bool
}

// ExtractArchive unpacks a tar.zst into dest. The archive may contain only
// regular files and directories (D-047). Symlinks, hard links and every other
// member type are rejected. A directory entry whose cleaned path is "." is
// the archive root, the entry `tar -C dir -cf … .` writes, not an escape.
// It rejects absolute paths, ".." escapes, device files and setuid/setgid/sticky
// modes. It does not follow symlinks and does not execute anything. On error
// the caller removes dest.
func ExtractArchive(r io.Reader, dest string, opt ExtractOptions) error {
	if dest == "" {
		return permanent(apierr.ValidationFailed, "extract destination")
	}
	dest = filepath.Clean(dest)
	for _, d := range opt.Deny {
		if pathsOverlap(dest, d) {
			return permanent(apierr.ValidationFailed, "extract destination overlaps persistent state")
		}
	}
	if opt.DiskFull {
		return transient(apierr.DiskSpaceLow, "extract")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		if fsutil.IsNoSpace(err) {
			return transient(apierr.DiskSpaceLow, "extract")
		}
		return err
	}
	dec, err := zstd.NewReader(r)
	if err != nil {
		return permanent(apierr.HashMismatch, "artifact is not zstd")
	}
	defer dec.Close()
	capBytes := opt.MaxBytes
	if capBytes <= 0 {
		capBytes = defaultDecompressedCap
	}
	maxEntries := opt.MaxEntries
	if maxEntries <= 0 {
		maxEntries = maxArchiveEntries
	}
	// File bytes are capped at capBytes. The stream cap adds header overhead
	// so an archive whose installed size equals MaxBytes still extracts.
	stream := decompressedStreamCap(capBytes, maxEntries)
	tr := tar.NewReader(io.LimitReader(dec, stream))
	var written int64
	var count int
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return permanent(apierr.ValidationFailed, "archive")
		}
		count++
		if count > maxEntries {
			return permanent(apierr.ValidationFailed, "archive entry cap")
		}
		if err := extractOne(tr, dest, hdr, opt.Deny, capBytes, &written); err != nil {
			return err
		}
	}
}

func extractOne(tr *tar.Reader, dest string, hdr *tar.Header, deny []string, capBytes int64, written *int64) error {
	name := hdr.Name
	if strings.Contains(name, "\x00") || strings.Contains(name, "\\") {
		return permanent(apierr.ValidationFailed, "archive path")
	}
	full, err := safeJoin(dest, name)
	if err != nil {
		return err
	}
	for _, d := range deny {
		if pathsOverlap(full, d) {
			return permanent(apierr.ValidationFailed, "archive path touches persistent state")
		}
	}
	if hdr.Mode&^0o777 != 0 {
		return permanent(apierr.ValidationFailed, "archive mode")
	}
	switch hdr.Typeflag {
	case tar.TypeDir:
		return mkdirSafe(dest, full, 0o755)
	case tar.TypeReg:
		if full == dest {
			return permanent(apierr.ValidationFailed, "archive path")
		}
		if hdr.Size < 0 {
			return permanent(apierr.SizeMismatch, "archive member size")
		}
		if capBytes > 0 && *written+hdr.Size > capBytes {
			return permanent(apierr.SizeMismatch, "archive expands past the manifest")
		}
		*written += hdr.Size
		return writeReg(tr, dest, full, hdr)
	default:
		// Symlinks, hard links, devices, fifos and sockets are rejected.
		// Release archives contain regular files and directories only (D-034a).
		return permanent(apierr.ValidationFailed, "archive member type")
	}
}

func writeReg(tr *tar.Reader, dest, full string, hdr *tar.Header) error {
	if err := mkdirSafe(dest, filepath.Dir(full), 0o755); err != nil {
		return err
	}
	if err := parentsReal(dest, full); err != nil {
		return err
	}
	mode := os.FileMode(hdr.Mode) & 0o777
	if mode == 0 {
		mode = 0o644
	}
	f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		if fsutil.IsNoSpace(err) {
			return transient(apierr.DiskSpaceLow, "extract")
		}
		return permanent(apierr.ValidationFailed, "archive member exists or is linked")
	}
	ok := false
	defer func() {
		if !ok {
			_ = f.Close()
			_ = os.Remove(full)
		}
	}()
	if _, err := io.Copy(f, io.LimitReader(tr, hdr.Size)); err != nil {
		if fsutil.IsNoSpace(err) {
			return transient(apierr.DiskSpaceLow, "extract")
		}
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func safeJoin(dest, name string) (string, error) {
	if name == "" || strings.Contains(name, "\x00") {
		return "", permanent(apierr.ValidationFailed, "archive path")
	}
	slashed := strings.ReplaceAll(name, "\\", "/")
	if path.IsAbs(slashed) || filepath.IsAbs(name) {
		return "", permanent(apierr.ValidationFailed, "absolute archive path")
	}
	clean := path.Clean(slashed)
	// "." is the archive root (`./` from `tar -C dir -cf … .`), not an escape.
	if clean == "." {
		return dest, nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", permanent(apierr.ValidationFailed, "archive path escapes")
	}
	full := filepath.Join(dest, filepath.FromSlash(clean))
	rel, err := filepath.Rel(dest, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", permanent(apierr.ValidationFailed, "archive path escapes")
	}
	return full, nil
}

func mkdirSafe(dest, dir string, mode os.FileMode) error {
	dir = filepath.Clean(dir)
	if dir == dest {
		return nil
	}
	rel, err := filepath.Rel(dest, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return permanent(apierr.ValidationFailed, "archive directory escapes")
	}
	cur := dest
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." || part == ".." {
			return permanent(apierr.ValidationFailed, "archive directory")
		}
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(cur, mode); err != nil {
				if fsutil.IsNoSpace(err) {
					return transient(apierr.DiskSpaceLow, "extract")
				}
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
			return permanent(apierr.ValidationFailed, "archive path follows a link")
		}
	}
	return nil
}

// parentsReal rejects a member whose parent is a symlink, so a tar cannot
// redirect a later file through a link it just wrote.
func parentsReal(dest, full string) error {
	dir := filepath.Dir(full)
	for {
		if dir == dest || len(dir) < len(dest) {
			return nil
		}
		fi, err := os.Lstat(dir)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return permanent(apierr.ValidationFailed, "archive path follows a link")
		}
		dir = filepath.Dir(dir)
	}
}

// decompressedStreamCap is MaxBytes plus per-member header overhead plus the
// tar trailer. The +1 lets a stream that lands exactly on the allowance
// finish; a longer stream ends as an unexpected EOF and is rejected.
// Formula: fileBytes + entries*archiveEntryOverhead + archiveTrailer + 1.
func decompressedStreamCap(fileBytes int64, entries int) int64 {
	if fileBytes < 0 {
		fileBytes = 0
	}
	if entries < 0 {
		entries = 0
	}
	overhead := int64(entries) * archiveEntryOverhead
	sum := saturatingAdd(saturatingAdd(fileBytes, overhead), archiveTrailer)
	return saturatingAdd(sum, 1)
}

func saturatingAdd(a, b int64) int64 {
	if a > (1<<63-1)-b {
		return 1<<63 - 1
	}
	return a + b
}

// pathsOverlap reports whether path and root are the same directory or one
// contains the other. An empty path does not overlap anything (D-034f).
func pathsOverlap(path, root string) bool {
	if path == "" || root == "" {
		return false
	}
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if path == "." || root == "." {
		return false
	}
	sep := string(filepath.Separator)
	return path == root || strings.HasPrefix(path, root+sep) || strings.HasPrefix(root, path+sep)
}
