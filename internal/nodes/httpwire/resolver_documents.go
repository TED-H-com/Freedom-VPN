package httpwire

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
)

func (h *Handlers) GetResolverDocument(ctx context.Context, r gen.GetResolverDocumentRequestObject) (gen.GetResolverDocumentResponseObject, error) {
	if err := h.ready(ctx); err != nil {
		return nil, err
	}
	p := principal(ctx)
	doc, err := h.Runtime.Nodes.ResolverDocumentForNode(ctx, h.Runtime.ResolverDocuments, p.NodeID, p.IncarnationID, r.DocumentId)
	if err != nil {
		return nil, err
	}
	return gen.GetResolverDocument200JSONResponse{Body: doc.Canonical, Headers: gen.GetResolverDocument200ResponseHeaders{
		ETag: `"sha256:` + doc.Reference.ContentSha256 + `"`, CacheControl: "private, max-age=0",
	}}, nil
}
