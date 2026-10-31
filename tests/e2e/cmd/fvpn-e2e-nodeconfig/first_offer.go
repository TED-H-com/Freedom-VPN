package main

// TEST ONLY observer at the actual agent transport. It never synthesizes an
// agent response or edits SQLite. A completed socket write alone is not proof
// of consumption; closure additionally requires exact local digest and ACK.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace"
	_ "modernc.org/sqlite"
)

const desiredHold = "/var/lib/vpn-node/e2e/hold-desired"
const offerArm = "/var/lib/vpn-node/e2e/first-offer-arm"
const offerEvidence = "/var/lib/vpn-node/e2e/first-offer.json"

func digestBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func readAccepted(ctx context.Context, path string) (*offertrace.Local, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(1000)")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var generation string
	var applied sql.NullString
	var body []byte
	err = db.QueryRowContext(ctx, `SELECT CAST(accepted_generation AS TEXT),CAST(applied_generation AS TEXT),envelope FROM desired WHERE id=1`).Scan(&generation, &applied, &body)
	if err != nil {
		return nil, err
	}
	out := &offertrace.Local{Accepted: generation, EnvelopeSHA256: digestBytes(body), At: time.Now().UTC()}
	if applied.Valid {
		out.Applied = &applied.String
	}
	return out, nil
}

type firstOfferObserver struct {
	mu                         sync.Mutex
	arm, hold, evidence, store string
}

func newFirstOfferObserver() *firstOfferObserver {
	return &firstOfferObserver{arm: offerArm, hold: desiredHold, evidence: offerEvidence, store: "/var/lib/vpn-node/state/agent.db"}
}
func (f *firstOfferObserver) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		desired := r.Method == http.MethodGet && r.URL.Path == "/api/v1/agent/desired-state"
		observation := r.Method == http.MethodPost && r.URL.Path == "/api/v1/agent/observations"
		if !desired && !observation {
			next.ServeHTTP(w, r)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if desired {
			if _, err := os.Stat(f.hold); err == nil {
				http.Error(w, "TEST ONLY desired-state hold", 503)
				return
			}
		}
		arm, err := os.ReadFile(f.arm)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		var ev offertrace.Evidence
		raw, _ := os.ReadFile(f.evidence)
		_ = json.Unmarshal(raw, &ev)
		if ev.CaptureID != string(arm) {
			ev = offertrace.Evidence{CaptureID: string(arm)}
		}
		var report struct {
			ID      string `json:"report_id"`
			Applied string `json:"applied_config_generation"`
		}
		if observation {
			body, e := io.ReadAll(io.LimitReader(r.Body, 4<<20))
			if e != nil {
				http.Error(w, "capture read failed", 502)
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))
			_ = json.Unmarshal(body, &report)
		}
		// Product response bounds are below 4 MiB; ResponseRecorder avoids changing
		// bytes, ETag or status. No content is persisted, only a digest and IDs.
		rr := httptest.NewRecorder()
		next.ServeHTTP(rr, r)
		for k, v := range rr.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rr.Code)
		n, writeErr := w.Write(rr.Body.Bytes())
		if desired && writeErr == nil {
			writeErr = http.NewResponseController(w).Flush()
		}
		at := time.Now().UTC()
		if desired && rr.Code == 200 && ev.First == nil {
			var doc struct {
				Signed struct {
					Generation string `json:"config_generation"`
				} `json:"signed"`
			}
			if json.Unmarshal(rr.Body.Bytes(), &doc) != nil || doc.Signed.Generation == "" {
				ev.Error = "desired envelope not decoded"
			} else {
				ev.First = &offertrace.Delivery{Generation: doc.Signed.Generation, BodySHA256: digestBytes(rr.Body.Bytes()), Bytes: n, At: at, WriteComplete: writeErr == nil && n == rr.Body.Len()}
			}
		}
		if observation && ev.First != nil {
			local, e := readAccepted(r.Context(), f.store)
			if e != nil {
				ev.Error = "local store unavailable"
			} else if local.Accepted == ev.First.Generation && local.EnvelopeSHA256 == ev.First.BodySHA256 {
				ev.Local = local
				if local.Applied != nil && *local.Applied == ev.First.Generation && report.Applied == ev.First.Generation && (rr.Code == 200 || rr.Code == 202) && ev.ACK == nil {
					ev.ACK = &offertrace.ACK{ReportID: report.ID, Generation: report.Applied, Status: rr.Code, At: at}
				}
			}
		}
		raw, err = json.Marshal(ev)
		if err == nil {
			if os.WriteFile(f.evidence+".tmp", raw, 0600) == nil {
				_ = os.Rename(f.evidence+".tmp", f.evidence)
			}
		}
	})
}
