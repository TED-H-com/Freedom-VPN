package updates

import (
	"strings"
	"testing"
)

func TestRedactStripsSecrets(t *testing.T) {
	in := `path /home/someone/key postgres://fvpn:pw@127.0.0.1/fvpn bearer abc "token": "one-time" -----BEGIN EC PRIVATE KEY-----
secret
-----END EC PRIVATE KEY-----`
	out := Redact(in)
	for _, part := range []string{"/home/someone", "postgres://", "bearer abc", "one-time", "BEGIN EC"} {
		if strings.Contains(out, part) {
			t.Fatalf("redaction left %q in %s", part, out)
		}
	}
}

func TestClassifyFirstInstallRollback(t *testing.T) {
	health := `{"dns_policy":"failed","identity_and_state":"passed","internet_only":"failed","processes":"passed","tunnel_probe":"failed"}`
	lab := ClassifyFirstInstallRollback(RollbackFacts{
		HealthJSON: health, ErrorCode: "ROLLBACK_FAILED", ErrorDetail: "ROLLBACK_FAILED",
		Events: "event stage=checking detail=\nevent stage=rolling_back detail=\nevent stage=recovery_required detail=",
	})
	if lab != FindingLabPrecondition {
		t.Fatalf("lab: %s", lab)
	}
	if got := ClassifyFirstInstallRollback(RollbackFacts{ErrorCode: "HASH_MISMATCH"}); got != FindingNotReproduced {
		t.Fatalf("hash: %s", got)
	}
	defect := ClassifyFirstInstallRollback(RollbackFacts{
		HealthJSON: health, ErrorCode: "ROLLBACK_FAILED", LastGood: "rel",
		Events: "stage=rollback detail=symlink",
	})
	if defect != FindingUpdaterDefect {
		t.Fatalf("defect: %s", defect)
	}
}
