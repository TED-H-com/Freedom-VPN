//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"time"
)

func (r *runner) scaleHook(ctx context.Context, admin *session) {
	rec := Record{ID: "native-scale-hook", Title: "Native scale hook smoke: distinct users and peers", Spec: []string{"§8", "§19 AT12", "SECOPS-10"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	wait, cancel := context.WithTimeout(ctx, 8*time.Minute)
	defer cancel()
	out, err := (&Stack{lab: r.lab}).ScaleNativeMembers(wait, BrowserSession{Client: admin.http, CSRF: admin.csrf}, r.planRevID, "scale", 2, 2)
	raw, _ := json.Marshal(out)
	rec.Observations = append(rec.Observations, Observation{Name: "counts", Detail: string(raw)}, Observation{Name: "scope", Detail: "2 users x 2 native peers validates the hook; 20 x 3 load is not executed or claimed. The same existing compose/network/image is used."})
	if err != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "error", Detail: err.Error()})
		return
	}
	if out.Users == 2 && out.Peers == 4 && out.IdentityMatches == 4 && out.Authorized == 4 {
		rec.Status = StatusPass
	}
}
