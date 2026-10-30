package httpwire

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
)

func (h *Handlers) CreateAdminGroup(ctx context.Context, r gen.CreateAdminGroupRequestObject) (gen.CreateAdminGroupResponseObject, error) {
	if err := h.ready(ctx); err != nil {
		return nil, err
	}
	tx, err := httpapi.Transaction(ctx)
	if err != nil {
		return nil, err
	}
	out, err := h.Runtime.Nodes.CreateGroupTx(ctx, tx, principal(ctx).Scope(), requestID(ctx), *r.Body)
	if err != nil {
		return nil, err
	}
	if err = result(ctx, "member_group", out.GroupId, 201, nil); err != nil {
		return nil, err
	}
	return gen.CreateAdminGroup201JSONResponse{Body: out}, nil
}

func (h *Handlers) ListAdminGroups(ctx context.Context, r gen.ListAdminGroupsRequestObject) (gen.ListAdminGroupsResponseObject, error) {
	if err := h.ready(ctx); err != nil {
		return nil, err
	}
	out, err := h.Runtime.Nodes.ListGroups(ctx, r.Params)
	return gen.ListAdminGroups200JSONResponse{Body: out}, err
}
