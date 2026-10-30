package httpwire

import (
	"encoding/json"
	"net/http"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
)

// NodeDetail's schema permits additive properties. D-056 evidence is projected
// here until the contracts owner adds its typed member to the generated model.
type nodeDetailProjection struct {
	api.NodeDetail
	DNSPolicyEvidence presence.Nullable[api.DnsPolicyEvidence] `json:"dns_policy_evidence"`
}

type nodeDetailResponse struct {
	body nodeDetailProjection
	etag string
}

func (r nodeDetailResponse) VisitGetNodeResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", r.etag)
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(r.body)
}
