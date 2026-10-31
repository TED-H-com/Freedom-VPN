package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace"
)

func TestSECOPS21DeliveredAcceptedAndACKAreSeparate(t *testing.T) {
	dir := t.TempDir()
	f := &firstOfferObserver{arm: filepath.Join(dir, "arm"), hold: filepath.Join(dir, "hold"), evidence: filepath.Join(dir, "evidence"), store: filepath.Join(dir, "agent.db")}
	body := `{"signed":{"config_generation":"16"}}`
	db, e := sql.Open("sqlite", f.store)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(`CREATE TABLE desired(id INTEGER,accepted_generation INTEGER,applied_generation INTEGER,envelope BLOB); INSERT INTO desired VALUES(1,15,15,?)`, body); e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{f.arm, f.hold} {
		if e = os.WriteFile(p, []byte("capture"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	calls := 0
	h := f.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == "GET" {
			_, _ = w.Write([]byte(body))
		} else {
			w.WriteHeader(202)
		}
	}))
	get := func() {
		t.Helper()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/v1/agent/desired-state", nil))
	}
	post := func(gen string) {
		t.Helper()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/agent/observations", strings.NewReader(`{"report_id":"test-report","applied_config_generation":"`+gen+`"}`)))
	}
	read := func() offertrace.Evidence {
		t.Helper()
		b, e := os.ReadFile(f.evidence)
		if e != nil {
			t.Fatal(e)
		}
		var v offertrace.Evidence
		if e = json.Unmarshal(b, &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	get()
	if calls != 0 {
		t.Fatal("held request reached upstream")
	}
	if e = os.Remove(f.hold); e != nil {
		t.Fatal(e)
	}
	get()
	v := read()
	if v.First == nil || !v.First.WriteComplete || v.First.Generation != "16" || v.Local != nil || v.ACK != nil {
		t.Fatalf("delivery alone: %+v", v)
	}
	post("16")
	if read().ACK != nil {
		t.Fatal("central receipt without local acceptance")
	}
	if _, e = db.Exec(`UPDATE desired SET accepted_generation=16,applied_generation=16`); e != nil {
		t.Fatal(e)
	}
	post("17")
	v = read()
	if v.Local == nil || v.ACK != nil {
		t.Fatal("later ACK credited to first offer")
	}
	if _, e = db.Exec(`UPDATE desired SET envelope='wrong'`); e != nil {
		t.Fatal(e)
	}
	post("16")
	if read().ACK != nil {
		t.Fatal("mismatched local bytes accepted")
	}
	if _, e = db.Exec(`UPDATE desired SET envelope=?`, body); e != nil {
		t.Fatal(e)
	}
	post("16")
	v = read()
	if v.ACK == nil || v.ACK.ReportID != "test-report" {
		t.Fatal("matching evidence missing")
	}
	first := *v.First
	body = `{"signed":{"config_generation":"18"}}`
	get()
	if *read().First != first {
		t.Fatal("first response overwritten")
	}
	if _, e = readAccepted(context.Background(), filepath.Join(dir, "absent.db")); e == nil {
		t.Fatal("missing database accepted")
	}
}
