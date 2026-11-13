//go:build integration

package httpwire_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func Test14_1_NodeResponsesUTC(t *testing.T) {
	local := time.Local
	time.Local = time.FixedZone("UTC-5", -5*60*60)
	t.Cleanup(func() { time.Local = local })

	f := setup(t)
	nodeID := uuid.New()
	created := time.Date(2026, time.October, 27, 23, 59, 59, 123456000, time.Local)
	lastSeen := created.Add(10*time.Hour + 654321*time.Microsecond)
	if _, err := f.r.Pool.Exec(t.Context(), `INSERT INTO nodes (id, created_at, last_seen_at) VALUES ($1, $2, $3)`, nodeID, created, lastSeen); err != nil {
		t.Fatal(err)
	}
	var storedCreated, storedLastSeen time.Time
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT created_at, last_seen_at FROM nodes WHERE id=$1`, nodeID).Scan(&storedCreated, &storedLastSeen); err != nil {
		t.Fatal(err)
	}
	if !storedCreated.Equal(created) || !storedLastSeen.Equal(lastSeen) {
		t.Fatal("stored node times differ from the written instants")
	}

	type nodeTimes struct {
		NodeID     uuid.UUID `json:"node_id"`
		LastSeenAt string    `json:"last_seen_at"`
		CreatedAt  string    `json:"created_at"`
	}
	for _, endpoint := range []string{"list", "detail"} {
		t.Run(endpoint, func(t *testing.T) {
			// Bind contract validation to this subtest so both endpoints run even
			// if one response violates the schema.
			sub := *f
			sub.t = t
			validateFollowup28Responses(t, &sub)
			var node nodeTimes
			if endpoint == "list" {
				out := decode[struct {
					Items []nodeTimes `json:"items"`
				}](t, sub.call(http.MethodGet, "/api/v1/admin/nodes", nil, nil, nil), http.StatusOK)
				if len(out.Items) != 1 {
					t.Fatalf("node list has %d items, want 1", len(out.Items))
				}
				node = out.Items[0]
			} else {
				node = decode[nodeTimes](t, sub.call(http.MethodGet, "/api/v1/admin/nodes/"+nodeID.String(), nil, nil, nil), http.StatusOK)
			}
			if node.NodeID != nodeID {
				t.Fatalf("node ID = %s, want %s", node.NodeID, nodeID)
			}
			for _, field := range []struct {
				name  string
				value string
				want  time.Time
			}{
				{"last_seen_at", node.LastSeenAt, storedLastSeen},
				{"created_at", node.CreatedAt, storedCreated},
			} {
				if !strings.HasSuffix(field.value, "Z") {
					t.Errorf("%s = %q, want UTC RFC 3339 ending in Z", field.name, field.value)
				}
				got, err := time.Parse(time.RFC3339Nano, field.value)
				if err != nil || !got.Equal(field.want) {
					t.Errorf("%s = %q, want stored instant %s (parse error: %v)", field.name, field.value, field.want.Format(time.RFC3339Nano), err)
				}
			}
		})
	}
}
