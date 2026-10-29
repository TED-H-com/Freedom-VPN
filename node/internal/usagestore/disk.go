package usagestore

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"syscall"
)

// footprint is the on-disk size of the SQLite main file, the WAL and the
// shared-memory index sidecar (§10.8). Indexes live inside the main file,
// so they are included there. Any other sidecar named "<db>*" is included.
func footprint(path string) (uint64, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	var sum uint64
	for _, e := range entries {
		name := e.Name()
		if name != base && !hasPrefix(name, base) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		if info.IsDir() {
			continue
		}
		n := info.Size()
		if n < 0 {
			continue
		}
		if sum > math.MaxUint64-uint64(n) {
			return math.MaxUint64, nil
		}
		sum += uint64(n)
	}
	return sum, nil
}

func hasPrefix(name, base string) bool {
	return len(name) > len(base) && name[:len(base)] == base
}

// freeSpace is the available bytes on the filesystem that holds path.
// A zero MinFreeBytes leaves this unused: the floor is 待實測 (§10.8) and
// this package does not invent a production value.
func freeSpace(path string) (uint64, error) {
	dir := path
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		dir = filepath.Dir(path)
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, fmt.Errorf("usagestore: statfs: %w", err)
	}
	if st.Bsize <= 0 {
		return 0, fmt.Errorf("usagestore: statfs: invalid block size")
	}
	bsize := uint64(st.Bsize)
	avail := uint64(st.Bavail)
	if bsize != 0 && avail > math.MaxUint64/bsize {
		return math.MaxUint64, nil
	}
	return avail * bsize, nil
}
