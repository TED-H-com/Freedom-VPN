package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestSEC01_LockReferenceSurvivesRecreation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock")
	original := []byte("TEST ONLY original lock content")
	must(t, os.WriteFile(path, original, 0600))
	ref, err := holdLock(path)
	must(t, err)
	defer ref.file.Close()
	if !ref.same(path) {
		t.Fatal("held file did not match its original path")
	}
	initial, err := ref.file.Stat()
	must(t, err)
	for i := 0; i < 64; i++ {
		must(t, os.Remove(path))
		if ref.same(path) {
			t.Fatal("missing path matched held file")
		}
		content := original
		if i%2 != 0 {
			content = []byte("TEST ONLY changed lock content")
		}
		must(t, os.WriteFile(path, content, 0600))
		current, err := os.Lstat(path)
		must(t, err)
		held, err := ref.file.Stat()
		must(t, err)
		if !os.SameFile(initial, held) || os.SameFile(held, current) || ref.same(path) {
			t.Fatal("recreated path matched live original reference")
		}
	}
	// The loop requires no particular allocation/reuse policy: the original
	// remains alive solely through the descriptor after the first unlink.
	must(t, ref.file.Close())
	if ref.same(path) {
		t.Fatal("closed reference accepted a checkpoint")
	}
}
func TestSEC01_LockReferenceDoesNotAccessContentOrLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	content := []byte("TEST ONLY lock reference boundary")
	must(t, os.WriteFile(path, content, 0600))
	before := manifest(t, filepath.Dir(path))
	ref, err := holdLock(path)
	must(t, err)
	defer ref.file.Close()
	fd := int(ref.file.Fd())
	flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	must(t, err)
	descriptorFlags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
	must(t, err)
	if flags&unix.O_PATH == 0 || flags&unix.O_NOFOLLOW == 0 || descriptorFlags&unix.FD_CLOEXEC == 0 {
		t.Fatal("reference is not non-following O_PATH/CLOEXEC")
	}
	if _, err := unix.Read(fd, make([]byte, 1)); !errors.Is(err, unix.EBADF) {
		t.Fatal("reference permitted a content read")
	}
	if _, err := unix.Write(fd, []byte("TEST ONLY")); !errors.Is(err, unix.EBADF) {
		t.Fatal("reference permitted a content write")
	}
	other, err := os.OpenFile(path, os.O_RDWR, 0)
	must(t, err)
	defer other.Close()
	must(t, syscall.Flock(int(other.Fd()), syscall.LOCK_EX|syscall.LOCK_NB))
	must(t, syscall.Flock(int(other.Fd()), syscall.LOCK_UN))
	if !ref.same(path) {
		t.Fatal("non-mutating use changed held identity")
	}
	after, err := os.ReadFile(path)
	must(t, err)
	if !bytes.Equal(content, after) {
		t.Fatal("lock content changed")
	}
	if got := manifest(t, filepath.Dir(path)); !reflect.DeepEqual(before, got) {
		t.Fatal("reference acquisition changed files")
	}
}
