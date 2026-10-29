package dbtest

import (
	"net/url"
	"testing"

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
