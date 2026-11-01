// Package operations models desired work separately from observed evidence
// (§14.5, §15.4, §18.3). External calls never run inside these transactions.
package operations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Step struct {
	Required     bool                    `json:"required"`
	Status       api.OperationStepStatus `json:"status"`
	Reason       string                  `json:"reason,omitempty"`
	ObservedRef  string                  `json:"observed_ref,omitempty"`
	ObservedAt   *time.Time              `json:"observed_at,omitempty"`
	ErrorCode    *api.ErrorCode          `json:"error_code,omitempty"`
	ErrorMessage *string                 `json:"error_message,omitempty"`
}

var (
	boundedPublicationEvidenceCauses = []string{
		"route_missing",
		"route_disabled",
		"route_ambiguous",
		"route_contract_mismatch",
		"route_peer_mismatch",
		"route_groups_mismatch",
		"group_evidence_unavailable",
		"no_distribution_groups",
		"identity_unconfirmed",
		"adapter_unavailable",
	}
	boundedPublicationErrorKinds = []string{
		"not_found",
		"conflict",
		"rate_limited",
		"unauthorized",
		"upstream",
		"unavailable",
		"invalid",
		"unconfirmed",
	}
	boundedStepExtraCauses = []string{
		"member_forwarding_unconfirmed",
		"desired_state_unavailable",
	}
)

// IsBoundedFailureCause reports whether cause is in the fixed admin operation
// failure vocabulary (exits.PublicationFailureCause plus node publication step causes).
func IsBoundedFailureCause(cause string) bool {
	if slices.Contains(boundedPublicationEvidenceCauses, cause) || slices.Contains(boundedStepExtraCauses, cause) {
		return true
	}
	if status, ok := strings.CutPrefix(cause, "publish_rejected:"); ok {
		if len(status) != 3 {
			return false
		}
		code, err := strconv.Atoi(status)
		return err == nil && code >= 400 && code <= 599 && strconv.Itoa(code) == status
	}
	if kind, ok := strings.CutPrefix(cause, "publish_failed:"); ok {
		return slices.Contains(boundedPublicationErrorKinds, kind)
	}
	if kind, ok := strings.CutPrefix(cause, "readback_failed:"); ok {
		return slices.Contains(boundedPublicationErrorKinds, kind)
	}
	return false
}

// PublicationGateMessage preserves both stored diagnostics in revision 13's
// OperationStep.message. The gate vocabulary comes from nodes.PublicationDataPathFailure.
// Both inputs must be bounded; their combined ASCII text fits maxLength 1000
// without truncating either value or admitting arbitrary stored/provider text.
func PublicationGateMessage(message, cause string) (string, bool) {
	if !IsBoundedFailureCause(message) {
		return "", false
	}
	switch cause {
	case "stale_generation", "check_failed", "forwarding_disabled", "tunnel_not_passed",
		"isolation_unconfirmed", "readiness_unconfirmed", "serving_checks_unconfirmed", "lease_unconfirmed":
		return message + " (" + cause + ")", true
	default:
		return "", false
	}
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

// GenerationHoldAlarm is central diagnostic evidence, not a serving grant or a
// new Operation status. Nullable counters stay unknown after a malformed summary.
type GenerationHoldAlarm struct {
	IncidentID        uuid.UUID  `json:"incident_id"`
	NodeID            uuid.UUID  `json:"node_id"`
	IncarnationID     uuid.UUID  `json:"incarnation_id"`
	ReportID          uuid.UUID  `json:"report_id"`
	BootID            uuid.UUID  `json:"boot_id"`
	LeaseID           uuid.UUID  `json:"lease_id"`
	FirstSeenAt       time.Time  `json:"first_seen_at"`
	LastSeenAt        time.Time  `json:"last_seen_at"`
	AppliedGeneration *string    `json:"applied_generation"`
	LeaseGeneration   string     `json:"lease_generation"`
	ReportedSteps     *string    `json:"reported_steps"`
	ReportedElapsedMS *string    `json:"reported_elapsed_ms"`
	ReportedAlarm     *bool      `json:"reported_alarm"`
	SummaryValid      bool       `json:"summary_valid"`
	AlarmReportID     *uuid.UUID `json:"alarm_report_id"`
}

// RecordGenerationHoldAlarm appends the existing admin audit surface. The caller
// locks the durable incident and marks escalated_at in this same transaction,
// so a rollback/retry or another replica cannot duplicate the escalation.
func RecordGenerationHoldAlarm(ctx context.Context, tx pgx.Tx, alarm GenerationHoldAlarm) error {
	raw, err := json.Marshal(alarm)
	if err != nil {
		return err
	}
	return db.AppendAudit(ctx, tx, db.AuditEvent{ID: uuid.Must(uuid.NewV7()), Actor: "generation-hold-monitor", ActorType: "system",
		Action: "node.generation_hold.escalated", Resource: "node:" + alarm.NodeID.String(), Result: "succeeded",
		SchemaVersion: 1, Before: []byte(`{}`), After: raw})
}

func (s Store) Get(ctx context.Context, id uuid.UUID, p auth.Principal) (op api.Operation, err error) {
	if p.Role != auth.Admin && p.Role != auth.Member {
		return op, apierr.New(apierr.Forbidden, "Role is not allowed.")
	}
	var raw []byte
	var version, desired, observed *string
	var rid *uuid.UUID
	var completedAt *time.Time
	var errorCode, errorMessage *string
	op.Error.SetNull()
	err = s.Pool.QueryRow(ctx, `SELECT id,kind,status,resource_type,resource_id,resource_version::text,desired_generation::text,observed_generation::text,steps,request_id,created_at,updated_at,completed_at,error_code,error_message FROM operations WHERE id=$1 AND ($2::boolean OR principal=$3)`, id, p.Role == auth.Admin, p.Scope()).Scan(&op.OperationId, &op.Kind, &op.Status, &op.Resource.Type, &op.Resource.Id, &version, &desired, &observed, &raw, &rid, &op.CreatedAt, &op.UpdatedAt, &completedAt, &errorCode, &errorMessage)
	if errors.Is(err, pgx.ErrNoRows) {
		return op, apierr.New(apierr.ResourceNotFound, "Operation not found.")
	}
	if err != nil {
		return op, err
	}
	if rid != nil {
		op.RequestId = *rid
	}
	for _, field := range []struct {
		str    *string
		target *presence.Nullable[decstr.U64]
	}{{version, &op.ResourceVersion}, {desired, &op.DesiredGeneration}, {observed, &op.ObservedGeneration}} {
		str, target := field.str, field.target
		target.SetNull()
		if str != nil {
			v, e := decstr.Parse(*str)
			if e != nil {
				return op, e
			}
			target.Set(v)
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
		step := api.OperationStep{Name: name, Required: v.Required, Status: v.Status, ObservedAt: presence.FromPointer(v.ObservedAt), ErrorCode: presence.FromPointer(v.ErrorCode),
			Message: presence.NewNullNullable[string]()}
		if v.Reason != "" {
			reason := v.Reason
			step.Message.Set(reason)
		}
		if v.ErrorCode != nil {
			message := "Step failed; quote the operation ID."
			if p.Role == auth.Admin && v.ErrorMessage != nil && IsBoundedFailureCause(*v.ErrorMessage) {
				message = *v.ErrorMessage
			}
			step.Message.Set(message)
		}
		if p.Role == auth.Admin && op.Kind == api.OperationKindNodePublish && v.ErrorMessage != nil {
			if message, ok := PublicationGateMessage(*v.ErrorMessage, v.Reason); ok {
				step.Message.Set(message)
			}
		}
		if (op.Status == "failed" || op.Status == "partially_succeeded") && v.ErrorCode != nil && op.Error.IsNull() {
			op.Error.Set(struct {
				Code    api.ErrorCode `json:"code"`
				Message string        `json:"message"`
			}{Code: *v.ErrorCode, Message: "Required operation step failed; quote the operation ID."})
		}
		op.Steps = append(op.Steps, step)
	}
	if (op.Status == "failed" || op.Status == "partially_succeeded") && op.Error.IsNull() && errorCode != nil {
		code := api.ErrorCode(*errorCode)
		if !code.Valid() {
			return op, errors.New("invalid stored operation error code")
		}
		message := failureMessage
		if errorMessage != nil && *errorMessage != "" {
			message = *errorMessage
		}
		op.Error.Set(struct {
			Code    api.ErrorCode `json:"code"`
			Message string        `json:"message"`
		}{code, message})
	}
	op.CreatedAt = op.CreatedAt.UTC()
	op.UpdatedAt = op.UpdatedAt.UTC()
	if completedAt != nil {
		t := completedAt.UTC()
		completedAt = &t
	}
	op.CompletedAt = presence.FromPointer(completedAt)
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
