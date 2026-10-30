package dbtest

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// new-style: fvpn_t<unix seconds hex>_<32 hex>
// legacy:    fvpn_<32 hex>
var (
	newNamePattern    = regexp.MustCompile(`^fvpn_t([0-9a-f]+)_([0-9a-f]{32})$`)
	legacyNamePattern = regexp.MustCompile(`^fvpn_[0-9a-f]{32}$`)
)

// databaseName is fvpn_t<unix seconds hex>_<32 hex>. The random part is a UUID
// with the dashes removed, the same 128-bit value dbtest used before timestamps.
// The result is always shorter than PostgreSQL's 63-byte identifier limit.
func databaseName(created time.Time) (string, error) {
	sec := created.Unix()
	if sec < 0 {
		return "", errors.New("invalid database creation time")
	}
	random := strings.ReplaceAll(uuid.NewString(), "-", "")
	if len(random) != 32 {
		return "", errors.New("invalid database name entropy")
	}
	name := "fvpn_t" + strconv.FormatInt(sec, 16) + "_" + random
	if len(name) >= 63 {
		return "", errors.New("database name exceeds 62 bytes")
	}
	return name, nil
}

// createdAt reports the creation time encoded in a new-style name.
func createdAt(name string) (time.Time, bool) {
	match := newNamePattern.FindStringSubmatch(name)
	if match == nil {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(match[1], 16, 64)
	if err != nil || sec < 0 {
		return time.Time{}, false
	}
	return time.Unix(sec, 0).UTC(), true
}

// sweepCandidate reports whether name may be dropped. New-style names must be
// strictly older than olderThan. Legacy names require legacy to be set and have
// no age. Any other name is rejected.
func sweepCandidate(name string, now time.Time, olderThan time.Duration, legacy bool) bool {
	if created, ok := createdAt(name); ok {
		return created.Before(now.Add(-olderThan))
	}
	return legacy && legacyNamePattern.MatchString(name)
}
