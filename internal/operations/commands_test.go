package operations

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestAPI01_CreateRequiresTransaction(t *testing.T) {
	r := CreateRequest{Kind: api.OperationKindNodeEnroll, Principal: auth.Principal{Role: auth.Admin, ID: uuid.New()}, Resource: api.ResourceRef{Type: api.ResourceTypeNode, Id: uuid.New()}, RequestID: uuid.New(), Steps: map[string]Step{"apply": {Required: true}}}
	id, err := Create(t.Context(), nil, r)
	if err == nil || id != uuid.Nil {
		t.Fatal("operation creation accepted a nil transaction", id, err)
	}
}
