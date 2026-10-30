package dbtest

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Test15_5_IsolatedDSN(t *testing.T) {
	const name = "fvpn_test_isolation"
	for _, dsn := range []string{
		"postgres://test@db.example.invalid/admin?sslmode=verify-full&application_name=cli%20test&connect_timeout=7",
		"postgresql://test@db.example.invalid/old%2Fdatabase?sslmode=disable&options=-c%20search_path%3Dpublic&application_name=a&application_name=b",
		"postgres:///admin?host=%2Fvar%2Frun%2Fpostgresql&sslmode=disable",
	} {
		isolated, err := isolatedDSN(dsn, name)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := url.Parse(dsn)
		after, _ := url.Parse(isolated)
		if after.Path != "/"+name || after.RawQuery != before.RawQuery || after.Host != before.Host || after.User.String() != before.User.String() || after.Scheme != before.Scheme {
			t.Fatal("isolated URL did not preserve connection settings")
		}
		cfg, err := pgxpool.ParseConfig(isolated)
		if err != nil {
			t.Fatal("parse isolated config failed")
		}
		// This is the serialization boundary used by subprocesses, not only the
		// in-memory Database field which hid the original CLI isolation defect.
		reopened, err := pgxpool.ParseConfig(cfg.ConnConfig.ConnString())
		if err != nil || reopened.ConnConfig.Database != name {
			t.Fatal("serialized config does not target isolated database")
		}
	}
	for _, dsn := range []string{
		"host=db.example.invalid dbname=admin", "https://db.example.invalid/admin",
		"postgres://db.example.invalid/%zz", "postgres://db.example.invalid/admin#fragment",
		"postgres://db.example.invalid/admin?dbname=admin",
		"postgres://db.example.invalid/admin?database=admin",
	} {
		if _, err := isolatedDSN(dsn, name); err == nil {
			t.Fatal("accepted unsafe test URL")
		}
	}
}

func TestDatabaseNameRoundTrip(t *testing.T) {
	created := time.Unix(1_700_000_000, 123).UTC()
	first, err := databaseName(created)
	if err != nil {
		t.Fatal(err)
	}
	second, err := databaseName(created)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("names reused random entropy")
	}
	if len(first) >= 63 || len(second) >= 63 {
		t.Fatalf("name length %d %d", len(first), len(second))
	}
	for _, name := range []string{first, second} {
		got, ok := createdAt(name)
		if !ok || !got.Equal(time.Unix(created.Unix(), 0).UTC()) {
			t.Fatalf("parsed %s as %v ok=%v", name, got, ok)
		}
		if legacyNamePattern.MatchString(name) {
			t.Fatalf("new name matched legacy pattern: %s", name)
		}
		random := strings.SplitN(name, "_", 3)[2]
		if len(random) != 32 {
			t.Fatalf("random entropy %q", random)
		}
	}
	if _, err := databaseName(time.Unix(-1, 0)); err == nil {
		t.Fatal("accepted a negative creation time")
	}
}

func TestSweepCandidate(t *testing.T) {
	created := time.Unix(1_700_000_000, 0).UTC()
	name, err := databaseName(created)
	if err != nil {
		t.Fatal(err)
	}
	legacy := "fvpn_" + strings.Repeat("ab", 16)
	now := created.Add(time.Hour)
	if sweepCandidate(name, now, time.Hour, false) {
		t.Fatal("name exactly one hour old was eligible")
	}
	if !sweepCandidate(name, now.Add(time.Second), time.Hour, false) {
		t.Fatal("name older than one hour was kept")
	}
	if sweepCandidate(name, created.Add(30*time.Minute), time.Hour, true) {
		t.Fatal("young name was eligible")
	}
	if sweepCandidate(legacy, now, time.Hour, false) {
		t.Fatal("legacy name was eligible without the flag")
	}
	if !sweepCandidate(legacy, now, time.Hour, true) {
		t.Fatal("legacy name was kept with the flag")
	}
	for _, foreign := range []string{
		"postgres", "template0", "template1", "fvpn_do_not_touch",
		"fvpn_", "fvpn_" + strings.Repeat("ab", 15), strings.ToUpper(legacy),
		"fvpn_t" + strings.ToUpper("65f00000") + "_" + strings.Repeat("ab", 16),
	} {
		if sweepCandidate(foreign, now, 0, true) {
			t.Fatalf("non-matching name was eligible: %s", foreign)
		}
	}
}
