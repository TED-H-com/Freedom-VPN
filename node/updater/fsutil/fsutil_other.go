//go:build !linux

package fsutil

import "errors"

var errUnsupported = errors.New("fsutil: only linux nodes are supported")

func device(string) (uint64, error) { return 0, errUnsupported }

// FreeBytes is only implemented on linux; nodes run Ubuntu (§3.2).
func FreeBytes(string) (uint64, error) { return 0, errUnsupported }
