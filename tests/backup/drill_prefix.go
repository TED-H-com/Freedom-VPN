package backup

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	drillPrefixFamily = "fvpn-w4-ops-backups"
	// Run prefixes are capped so derived container and volume names stay short.
	drillPrefixMaxLen = 63
)

// drillPrefixRE requires a run-specific suffix. The bare family name is rejected
// so cleanup cannot match every fvpn-w4-ops-backups-* object on the host.
var drillPrefixRE = regexp.MustCompile(`^fvpn-w4-ops-backups(?:-[a-z0-9]+)+$`)

// drillNames are the Docker objects one PITR drill run is allowed to create
// and remove. Every name is derived from Prefix.
type drillNames struct {
	Prefix           string
	Image            string
	PrimaryContainer string
	PrimaryVolume    string
	RepoVolume       string
	RestoreContainer string
	RestoreVolume    string
	VerifyPrefix     string
	Network          string
}

// deriveDrillPrefix returns envValue when it is a run-scoped name under
// fvpn-w4-ops-backups-. Otherwise it returns fvpn-w4-ops-backups- plus 8
// lowercase hex characters read from randomSource (4 bytes).
func deriveDrillPrefix(envValue string, randomSource io.Reader) (string, error) {
	if envValue != "" {
		if err := validateDrillPrefix(envValue); err != nil {
			return "", err
		}
		return envValue, nil
	}
	if randomSource == nil {
		return "", fmt.Errorf("backup: drill prefix entropy source is nil")
	}
	var buf [4]byte
	if _, err := io.ReadFull(randomSource, buf[:]); err != nil {
		return "", fmt.Errorf("backup: read drill prefix entropy: %w", err)
	}
	return drillPrefixFamily + "-" + hex.EncodeToString(buf[:]), nil
}

// drillNamesFromEnv resolves FVPN_BACKUP_DRILL_PREFIX. A nil randomSource uses
// crypto/rand, which is what the integration drill uses when the variable is unset.
func drillNamesFromEnv(envValue string, randomSource io.Reader) (drillNames, error) {
	if envValue == "" && randomSource == nil {
		randomSource = cryptorand.Reader
	}
	prefix, err := deriveDrillPrefix(envValue, randomSource)
	if err != nil {
		return drillNames{}, err
	}
	return drillNamesForPrefix(prefix)
}

func drillNamesForPrefix(prefix string) (drillNames, error) {
	if err := validateDrillPrefix(prefix); err != nil {
		return drillNames{}, err
	}
	return drillNames{
		Prefix:           prefix,
		Image:            prefix + "-postgres:17",
		PrimaryContainer: prefix + "-primary",
		PrimaryVolume:    prefix + "-primary-pgdata",
		RepoVolume:       prefix + "-repo",
		RestoreContainer: prefix + "-restore",
		RestoreVolume:    prefix + "-restore-pgdata",
		VerifyPrefix:     prefix + "-verify",
		Network:          prefix + "-net",
	}, nil
}

func validateDrillPrefix(prefix string) error {
	if len(prefix) > drillPrefixMaxLen || !drillPrefixRE.MatchString(prefix) {
		return fmt.Errorf("backup: invalid FVPN_BACKUP_DRILL_PREFIX %q (want %s-<lowercase-run-id>, at most %d characters)", prefix, drillPrefixFamily, drillPrefixMaxLen)
	}
	return nil
}

// ownedByDrillPrefix reports whether name is the run prefix itself or an object
// created under it (name equals prefix, or name starts with prefix + "-").
// A neighboring run whose id merely shares a leading hex string does not match.
func ownedByDrillPrefix(name, prefix string) bool {
	if validateDrillPrefix(prefix) != nil {
		return false
	}
	name = strings.TrimPrefix(strings.TrimSpace(name), "/")
	if name == "" || strings.ContainsAny(name, " \t\r\n") || strings.HasPrefix(name, "-") {
		return false
	}
	return name == prefix || strings.HasPrefix(name, prefix+"-")
}
