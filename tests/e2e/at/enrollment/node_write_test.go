//go:build e2e

package enrollment

import (
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

func (p *probe) writeServingCapacity(node uuid.UUID, detail map[string]any) labstack.NodeWriteResult {
	body := map[string]any{"capacity": map[string]any{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}
	return p.writeNode(node, "PATCH", "", body, labstack.NodeFieldsMatch(body), detail, 200, 202)
}

func (p *probe) writePublication(node, group uuid.UUID, reason string, detail map[string]any) labstack.NodeWriteResult {
	body := api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, GroupIds: &[]uuid.UUID{group}, Reason: reason}
	return p.writeNode(node, "PUT", "/publication", body, labstack.NodePublished(group), detail, 200, 202)
}

// Every caller supplies its check's detail, so failure and already-intended
// receipts survive the same serialization as successful command receipts.
func (p *probe) writeNode(node uuid.UUID, method, suffix string, body any, intended func([]byte) bool, detail map[string]any, success ...int) (result labstack.NodeWriteResult) {
	defer func() {
		row := safeResponse(result.Reply, result.Err)
		row["node_write"] = result.Receipt
		row["already_intended"] = result.AlreadyIntended
		detail[method+suffix] = row
	}()
	path := "/api/v1/admin/nodes/" + node.String()
	read, err := p.call("GET", path, nil)
	p.keep(read)
	if err != nil || read.Status != 200 {
		if err == nil {
			err = errors.New("initial node read failed")
		}
		return labstack.NodeWriteResult{Reply: read, Err: err}
	}
	result = p.stack.WriteNode(p.ctx, p.admin, method, path+suffix, path, body, read.Header.Get("ETag"), intended, success...)
	p.keep(result.Reply)
	return result
}

// Already-intended keeps the real 412 and supplies no operation. Resolve the
// existing revocation through audit evidence since this incarnation's retirement;
// ambiguous, incomplete or missing evidence cannot replace worker read-back.
func (p *probe) revokeOperation(node uuid.UUID, result labstack.NodeWriteResult, retiredAt time.Time, detail map[string]any) (uuid.UUID, bool) {
	if !result.Accepted(202) {
		return uuid.Nil, false
	}
	if !result.AlreadyIntended {
		var accepted api.OperationAccepted
		ok := json.Unmarshal(result.Reply.Body, &accepted) == nil && accepted.OperationId != uuid.Nil
		return accepted.OperationId, ok
	}
	if retiredAt.IsZero() {
		return uuid.Nil, false
	}
	path := "/api/v1/admin/audit-events?resource_type=node&resource_id=" + node.String() + "&occurred_from=" + url.QueryEscape(retiredAt.UTC().Format(time.RFC3339Nano)) + "&limit=100"
	r, err := p.call("GET", path, nil)
	p.keep(r)
	detail["revoke_operation_lookup"] = safeResponse(r, err)
	var page api.AuditEventList
	if err != nil || r.Status != 200 || json.Unmarshal(r.Body, &page) != nil || !page.NextCursor.IsNull() {
		return uuid.Nil, false
	}
	id := uuid.Nil
	for _, event := range page.Items {
		if event.Action != "node.revoke" || event.Result != "succeeded" || event.Resource.Type != "node" || event.Resource.Id != node || event.OccurredAt.Before(retiredAt) {
			continue
		}
		candidate, e := event.OperationId.Get()
		if e != nil || candidate == uuid.Nil || id != uuid.Nil {
			return uuid.Nil, false
		}
		id = candidate
	}
	detail["revoke_existing_operation_id"] = id
	return id, id != uuid.Nil
}
