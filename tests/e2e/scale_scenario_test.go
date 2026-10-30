//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"time"
)

func (r *runner) scaleHook(ctx context.Context, admin *session) {
	rec := Record{ID: "native-scale-hook", Title: "60 simultaneous native peers: gateway budget and protected streams", Spec: []string{"§8", "§19 AT12", "SECOPS-10"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	wait, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	out, err := (&Stack{lab: r.lab}).ScaleNativeMembers(wait, BrowserSession{Client: admin.http, CSRF: admin.csrf}, r.planRevID, "scale", 20, 3, func(ctx context.Context) (BrowserSession, error) {
		return (&Stack{lab: r.lab}).Login(ctx, "e2e-admin", "admin")
	})
	raw, _ := json.Marshal(out)
	rec.Observations = append(rec.Observations, Observation{Name: "counts", Detail: string(raw)}, Observation{Name: "scope", Detail: "20 users x 3 native peers on the existing compose/network/image; each identity and authorized device checked. Capacity is a lab budget, not a production hardware claim."})
	if err != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "error", Detail: err.Error()})
		return
	}
	if out.Users == 20 && out.Peers == 60 && out.IdentityMatches == 60 && out.Authorized == 60 && out.ScannedLogs == 60 && out.ProtectedLogs == 60 && out.SimultaneousAuthorized == 60 {
		rec.Status = StatusPass
	}
}
