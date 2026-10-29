// Package operations models desired work separately from observed evidence
// (§14.5, §15.4, §18.3). External calls never run inside these transactions.
package operations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Step struct {
	Required    bool                    `json:"required"`
	Status      api.OperationStepStatus `json:"status"`
	Reason      string                  `json:"reason,omitempty"`
	ObservedRef string                  `json:"observed_ref,omitempty"`
	ObservedAt  *time.Time              `json:"observed_at,omitempty"`
	ErrorCode   *api.ErrorCode          `json:"error_code,omitempty"`
}
type Evidence struct {
	Reference  string    `json:"reference"`
	ObservedAt time.Time `json:"observed_at"`
}
type CreateRequest struct {
	Kind                               api.OperationKind
	Principal                          auth.Principal
	Resource                           api.ResourceRef
	RequestID                          uuid.UUID
	ResourceVersion, DesiredGeneration *decstr.U64
	Steps                              map[string]Step
}
type Store struct{ Pool *pgxpool.Pool }

// Create must share the domain transaction with desired state, audit and outbox.
func Create(ctx context.Context, tx pgx.Tx, r CreateRequest) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	validPrincipal := ((r.Principal.Role == auth.Admin || r.Principal.Role == auth.Member) && r.Principal.ID != uuid.Nil) || (r.Principal.Role == auth.Agent && r.Principal.NodeID != uuid.Nil && r.Principal.IncarnationID != uuid.Nil)
	if len(r.Steps) < 1 || len(r.Steps) > 64 || r.Resource.Id == uuid.Nil || r.RequestID == uuid.Nil || !validPrincipal {
		return uuid.Nil, errors.New("invalid operation")
	}
	validKinds := []string{"node.update", "node.publish", "node.unpublish", "node.disable", "node.revoke", "node.retire", "member.update", "member.suspend", "member.unsuspend", "member.grant", "member.device_revoke", "policy.activate", "release.register", "update_batch.create", "update_batch.retry", "checkout.create"}
	if !slices.Contains(validKinds, string(r.Kind)) {
		return uuid.Nil, errors.New("invalid operation kind")
	}
	steps := make(map[string]Step, len(r.Steps))
	for name, s := range r.Steps {
		if !regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`).MatchString(name) {
			return uuid.Nil, errors.New("invalid step name")
		}
		if s.Required {
			s = Step{Required: true, Status: "pending"}
		} else {
			if strings.TrimSpace(s.Reason) == "" || len(s.Reason) > 1000 {
				return uuid.Nil, errors.New("not-required step needs a reason")
			}
			s = Step{Status: "not_required", Reason: s.Reason}
		}
		steps[name] = s
	}
	status := derive(steps)
	refs := map[string]Evidence{}
	var completed *time.Time
	if status == "succeeded" {
		now := time.Now().UTC()
		completed = &now
		refs["disposition"] = Evidence{Reference: "not_required", ObservedAt: now}
	}
	body, _ := json.Marshal(steps)
	evidence, _ := json.Marshal(refs)
	_, e := tx.Exec(ctx, `INSERT INTO operations(id,kind,principal,resource_type,resource_id,request_id,resource_version,desired_generation,steps,observed_refs,status,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`, id, r.Kind, r.Principal.Scope(), r.Resource.Type, r.Resource.Id, r.RequestID, decimal(r.ResourceVersion), decimal(r.DesiredGeneration), body, evidence, status, completed)
	return id, e
}
func decimal(v *decstr.U64) any {
	if v == nil {
		return nil
	}
	return v.String()
}

// Transition accepts evidence references only; the domain must obtain and persist
// actual read-back evidence before calling it. A submitted request is not evidence.
func Transition(ctx context.Context, tx pgx.Tx, id uuid.UUID, name string, status api.OperationStepStatus, evidence *Evidence, code *api.ErrorCode, observed *decstr.U64) error {
	var raw, refsRaw []byte
	var old api.OperationStatus
	var prior, desired *string
	err := tx.QueryRow(ctx, `SELECT steps,observed_refs,status,observed_generation::text,desired_generation::text FROM operations WHERE id=$1 FOR UPDATE`, id).Scan(&raw, &refsRaw, &old, &prior, &desired)
	if err != nil {
		return err
	}
	if old == "succeeded" || old == "superseded" {
		return apierr.New(apierr.StateConflict, "Operation is terminal.")
	}
	steps := map[string]Step{}
	refs := map[string]Evidence{}
	if err = json.Unmarshal(raw, &steps); err != nil {
		return err
	}
	if err = json.Unmarshal(refsRaw, &refs); err != nil {
		return err
	}
	s, ok := steps[name]
	if !ok || !s.Required {
		return errors.New("unknown or not-required step")
	}
	allowed := (s.Status == "pending" && (status == "running" || status == "succeeded" || status == "failed" || status == "cancelled")) || (s.Status == "running" && (status == "succeeded" || status == "failed" || status == "cancelled")) || (s.Status == "failed" && status == "running")
	if !allowed {
		return apierr.New(apierr.StateConflict, "Invalid step transition.")
	}
	if status == "succeeded" {
		if evidence == nil || !regexp.MustCompile(`^[a-z][a-z0-9_]*:[0-9a-f-]{36}$`).MatchString(evidence.Reference) || evidence.ObservedAt.IsZero() {
			return errors.New("successful step requires persisted evidence reference")
		}
		evidence.ObservedAt = evidence.ObservedAt.UTC()
		s.ObservedRef = name
		s.ObservedAt = &evidence.ObservedAt
		refs[name] = *evidence
	} else {
		s.ObservedRef = ""
		s.ObservedAt = nil
		delete(refs, name)
	}
	if status == "failed" && (code == nil || !apierr.Code(*code).Valid()) {
		return errors.New("failed step needs safe error code")
	}
	if status != "failed" {
		code = nil
	}
	s.Status = status
	s.ErrorCode = code
	steps[name] = s
	if observed != nil {
		if prior != nil && *observed < decstr.MustParse(*prior) {
			return errors.New("observed generation cannot decrease")
		}
		if desired != nil && *observed > decstr.MustParse(*desired) {
			return errors.New("observed generation exceeds operation intent")
		}
	}
	next := derive(steps)
	var completed *time.Time
	if next != "pending" && next != "running" {
		now := time.Now().UTC()
		completed = &now
	}
	// The database guard requires a running state before success/partial success.
	if (old == "pending" && next != "pending" && next != "failed" && next != "superseded") || ((old == "failed" || old == "partially_succeeded") && next != old) {
		if _, err = tx.Exec(ctx, `UPDATE operations SET status='running',completed_at=NULL WHERE id=$1`, id); err != nil {
			return err
		}
	}
	raw, _ = json.Marshal(steps)
	refsRaw, _ = json.Marshal(refs)
	_, err = tx.Exec(ctx, `UPDATE operations SET steps=$2,observed_refs=$3,status=$4,completed_at=$5,updated_at=clock_timestamp(),observed_generation=coalesce($6::numeric,observed_generation) WHERE id=$1`, id, raw, refsRaw, next, completed, decimal(observed))
	return err
}
func derive(steps map[string]Step) api.OperationStatus {
	pending, running, success, failed, cancelled := 0, 0, 0, 0, 0
	for _, s := range steps {
		if !s.Required {
			continue
		}
		switch s.Status {
		case "pending":
			pending++
		case "running":
			running++
		case "succeeded":
			success++
		case "failed":
			failed++
		case "cancelled":
			cancelled++
		}
	}
	if running > 0 || pending > 0 && (success+failed+cancelled) > 0 {
		return "running"
	}
	if pending > 0 {
		return "pending"
	}
	if cancelled > 0 {
		return "superseded"
	}
	if failed > 0 {
		if success > 0 {
			return "partially_succeeded"
		}
		return "failed"
	}
	return "succeeded"
}
func (s Store) Get(ctx context.Context, id uuid.UUID, p auth.Principal) (op api.Operation, err error) {
	if p.Role != auth.Admin && p.Role != auth.Member {
		return op, apierr.New(apierr.Forbidden, "Role is not allowed.")
	}
	var raw []byte
	var version, desired, observed *string
	var rid *uuid.UUID
	err = s.Pool.QueryRow(ctx, `SELECT id,kind,status,resource_type,resource_id,resource_version::text,desired_generation::text,observed_generation::text,steps,request_id,created_at,updated_at,completed_at FROM operations WHERE id=$1 AND ($2::boolean OR principal=$3)`, id, p.Role == auth.Admin, p.Scope()).Scan(&op.OperationId, &op.Kind, &op.Status, &op.Resource.Type, &op.Resource.Id, &version, &desired, &observed, &raw, &rid, &op.CreatedAt, &op.UpdatedAt, &op.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return op, apierr.New(apierr.ResourceNotFound, "Operation not found.")
	}
	if err != nil {
		return op, err
	}
	if rid != nil {
		op.RequestId = *rid
	}
	for str, target := range map[*string]**decstr.U64{version: &op.ResourceVersion, desired: &op.DesiredGeneration, observed: &op.ObservedGeneration} {
		if str != nil {
			v, e := decstr.Parse(*str)
			if e != nil {
				return op, e
			}
			*target = &v
		}
	}
	var steps map[string]Step
	if err = json.Unmarshal(raw, &steps); err != nil {
		return op, err
	}
	names := make([]string, 0, len(steps))
	for name := range steps {
		names = append(names, name)
	}
	slices.Sort(names)
	op.Steps = make([]api.OperationStep, 0, len(names))
	for _, name := range names {
		v := steps[name]
		step := api.OperationStep{Name: name, Required: v.Required, Status: v.Status, ObservedAt: v.ObservedAt, ErrorCode: v.ErrorCode}
		if v.Reason != "" {
			reason := v.Reason
			step.Message = &reason
		}
		if v.ErrorCode != nil {
			message := "Step failed; quote the operation ID."
			step.Message = &message
		}
		if (op.Status == "failed" || op.Status == "partially_succeeded") && v.ErrorCode != nil && op.Error == nil {
			op.Error = &struct {
				Code    api.ErrorCode `json:"code"`
				Message string        `json:"message"`
			}{Code: *v.ErrorCode, Message: "Required operation step failed; quote the operation ID."}
		}
		op.Steps = append(op.Steps, step)
	}
	op.CreatedAt = op.CreatedAt.UTC()
	op.UpdatedAt = op.UpdatedAt.UTC()
	if op.CompletedAt != nil {
		t := op.CompletedAt.UTC()
		op.CompletedAt = &t
	}
	return op, nil
}
func (s Store) GetAdminOperation(ctx context.Context, r gen.GetAdminOperationRequestObject) (gen.GetAdminOperationResponseObject, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Admin {
		return nil, apierr.New(apierr.Forbidden, "Administrator required.")
	}
	op, e := s.Get(ctx, r.OperationId, p)
	if e != nil {
		return nil, e
	}
	return gen.GetAdminOperation200JSONResponse{Body: op}, nil
}
func (s Store) GetMemberOperation(ctx context.Context, r gen.GetMemberOperationRequestObject) (gen.GetMemberOperationResponseObject, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Member {
		return nil, apierr.New(apierr.Forbidden, "Member required.")
	}
	op, e := s.Get(ctx, r.OperationId, p)
	if e != nil {
		return nil, e
	}
	return gen.GetMemberOperation200JSONResponse{Body: op}, nil
}
func Accepted(op api.Operation, role auth.Role) gen.OperationAcceptedJSONResponse {
	group := "member"
	if role == auth.Admin {
		group = "admin"
	}
	location := "/api/v1/" + group + "/operations/" + op.OperationId.String()
	return gen.OperationAcceptedJSONResponse{Body: op, Headers: gen.OperationAcceptedResponseHeaders{Location: &location}}
}
func WriteAccepted(w http.ResponseWriter, op api.Operation, role auth.Role) {
	r := Accepted(op, role)
	w.Header().Set("Location", *r.Headers.Location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(202)
	_ = json.NewEncoder(w).Encode(op)
}

// InTx is a convenience for worker step transitions, separate from HTTP commands.
func (s Store) InTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return db.InTx(ctx, s.Pool, db.TxOptions{}, fn)
}
