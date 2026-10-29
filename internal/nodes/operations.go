package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SystemActorUUID is the canonical D-029 UUID actor for system-initiated audit events.
var SystemActorUUID = uuid.MustParse("00000000-0000-0000-0000-000000000003")

// ResolveAdminActor resolves a caller-supplied principal or authenticated context
// into a D-029 conformant actor UUID, actor_type ("admin" or "system"), and non-blank principal scope.
func ResolveAdminActor(ctx context.Context, raw string) (uuid.UUID, string, string) {
	trimmed := strings.TrimSpace(raw)
	if p, ok := auth.FromContext(ctx); ok && p.ID != uuid.Nil {
		actorType := "admin"
		if p.Role == auth.Agent {
			actorType = string(api.AuditEventActorTypeNode)
		} else if p.Role != "" {
			actorType = string(p.Role)
		}
		if trimmed == "" {
			trimmed = p.Scope()
		}
		return p.ID, actorType, trimmed
	}
	if s := strings.TrimPrefix(trimmed, "admin:"); s != trimmed {
		if parsed, err := uuid.Parse(strings.TrimSpace(s)); err == nil && parsed != uuid.Nil {
			return parsed, "admin", trimmed
		}
	}
	if parsed, err := uuid.Parse(trimmed); err == nil && parsed != uuid.Nil {
		return parsed, "admin", trimmed
	}
	if trimmed == "system" {
		return SystemActorUUID, "system", "system"
	}
	if trimmed == "" {
		trimmed = "admin"
	}
	derived := uuid.NewSHA1(uuid.NameSpaceOID, []byte("fvpn-actor:"+trimmed))
	return derived, "admin", trimmed
}

// Querier abstracts pgxpool.Pool and pgx.Tx for database queries.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Standard publication/node operation step names (§5.2, §14.2, §18.3).
const (
	StepAgentPolicy          = "agent_policy"
	StepNetBirdRouteReadback = "netbird_route_readback"
	StepDataPathProbe        = "data_path_probe"
)

// Standard operation kinds for nodes and exits (§5.2, §14.2).
const (
	OpKindPublish   = string(api.OperationKindNodePublish)
	OpKindUnpublish = string(api.OperationKindNodeUnpublish)
	OpKindDisable   = string(api.OperationKindNodeDisable)
	OpKindUpdate    = string(api.OperationKindNodeUpdate)
	OpKindRevoke    = string(api.OperationKindNodeRevoke)
	OpKindRetire    = string(api.OperationKindNodeRetire)
)

// StoredStep represents the JSONB structure persisted in operations.steps[name],
// compatible with the database trigger guard_operation() (§15.4).
type StoredStep struct {
	Order        int        `json:"order"`
	Required     bool       `json:"required"`
	Status       string     `json:"status"`
	ObservedRef  string     `json:"observed_ref,omitempty"`
	ObservedAt   *time.Time `json:"observed_at,omitempty"`
	Reason       string     `json:"reason,omitempty"`
	ErrorCode    *string    `json:"error_code,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// NewNodeOperationSteps constructs the initial step map for a node/exit operation.
func NewNodeOperationSteps(kind string, now time.Time) map[string]StoredStep {
	now = now.UTC()
	switch kind {
	case OpKindPublish:
		return map[string]StoredStep{
			StepAgentPolicy: {
				Order:     1,
				Required:  true,
				Status:    string(api.OperationStepStatusPending),
				UpdatedAt: now,
			},
			StepNetBirdRouteReadback: {
				Order:     2,
				Required:  true,
				Status:    string(api.OperationStepStatusPending),
				UpdatedAt: now,
			},
			StepDataPathProbe: {
				Order:     3,
				Required:  true,
				Status:    string(api.OperationStepStatusPending),
				UpdatedAt: now,
			},
		}
	case OpKindUnpublish, OpKindDisable, OpKindRevoke, OpKindRetire:
		return map[string]StoredStep{
			StepAgentPolicy: {
				Order:     1,
				Required:  true,
				Status:    string(api.OperationStepStatusPending),
				UpdatedAt: now,
			},
			StepNetBirdRouteReadback: {
				Order:     2,
				Required:  true,
				Status:    string(api.OperationStepStatusPending),
				UpdatedAt: now,
			},
			StepDataPathProbe: {
				Order:     3,
				Required:  false,
				Status:    string(api.OperationStepStatusNotRequired),
				Reason:    "route withdrawn; data path probe not applicable",
				UpdatedAt: now,
			},
		}
	default: // OpKindUpdate
		return map[string]StoredStep{
			StepAgentPolicy: {
				Order:     1,
				Required:  true,
				Status:    string(api.OperationStepStatusPending),
				UpdatedAt: now,
			},
		}
	}
}

// CreateOperationParams holds parameters for inserting a new node operation row.
type CreateOperationParams struct {
	ID                uuid.UUID
	Kind              string
	Principal         string
	ResourceType      string
	ResourceID        uuid.UUID
	ResourceVersion   *string
	RequestID         *uuid.UUID
	DesiredGeneration *uint64
	Now               time.Time
}

// CreateOperation inserts a new pending operation in tx and returns its api.Operation representation.
func CreateOperation(ctx context.Context, tx pgx.Tx, p CreateOperationParams) (api.Operation, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := p.Now.UTC()
	if p.Now.IsZero() {
		now = time.Now().UTC()
	}
	if p.ResourceType == "" {
		p.ResourceType = string(api.ResourceTypeNode)
	}
	if !api.OperationKind(p.Kind).Valid() {
		return api.Operation{}, fmt.Errorf("nodes: invalid operation kind %q", p.Kind)
	}
	if !api.ResourceType(p.ResourceType).Valid() {
		return api.Operation{}, fmt.Errorf("nodes: invalid operation resource_type %q", p.ResourceType)
	}
	if p.Principal == "" {
		p.Principal = "system"
	}
	reqID := uuid.New()
	if p.RequestID != nil && *p.RequestID != uuid.Nil {
		reqID = *p.RequestID
	}
	resVer := "1"
	if p.ResourceVersion != nil && *p.ResourceVersion != "" {
		resVer = *p.ResourceVersion
	}

	steps := NewNodeOperationSteps(p.Kind, now)
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return api.Operation{}, err
	}
	var desiredGen any
	if p.DesiredGeneration != nil {
		desiredGen = strconv.FormatUint(*p.DesiredGeneration, 10)
	}
	const q = `
		INSERT INTO operations (
			id, kind, principal, resource_type, resource_id, request_id,
			desired_generation, resource_version, status, steps, observed_refs,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7::numeric, $8::numeric, 'pending', $9::jsonb, '{}'::jsonb,
			$10, $10
		)`
	if _, err := tx.Exec(ctx, q,
		p.ID, p.Kind, p.Principal, p.ResourceType, p.ResourceID, reqID,
		desiredGen, resVer, stepsJSON, now,
	); err != nil {
		return api.Operation{}, err
	}
	return GetOperationTx(ctx, tx, p.ID)
}

// GetOperation reads an operation by ID through the Service pool.
func (s *Service) GetOperation(ctx context.Context, opID uuid.UUID) (api.Operation, error) {
	return GetOperation(ctx, s.pool, opID)
}

// GetOperation reads an operation by ID using a pool or connection.
func GetOperation(ctx context.Context, q Querier, opID uuid.UUID) (api.Operation, error) {
	return scanOperation(ctx, q, opID)
}

// GetOperationTx reads an operation by ID inside a transaction.
func GetOperationTx(ctx context.Context, tx pgx.Tx, opID uuid.UUID) (api.Operation, error) {
	return scanOperation(ctx, tx, opID)
}

// ListMemberOperations returns all operations associated with memberID, following
// operations.resource_type = 'member' (directly), operations.resource_type = 'entitlement_grant'
// through entitlement_grants.member_id, and operations.resource_type = 'member_device'
// through member_devices.member_id.
func (s *Service) ListMemberOperations(ctx context.Context, memberID uuid.UUID) ([]api.Operation, error) {
	return ListMemberOperations(ctx, s.pool, memberID)
}

// ListMemberOperations queries operations associated with memberID across member,
// entitlement_grant (via entitlement_grants.member_id), and member_device (via member_devices.member_id).
func ListMemberOperations(ctx context.Context, q Querier, memberID uuid.UUID) ([]api.Operation, error) {
	const sqlQuery = `
		SELECT o.id
		FROM operations o
		WHERE (o.resource_type = 'member' AND o.resource_id = $1)
		   OR (o.resource_type = 'entitlement_grant' AND o.resource_id IN (SELECT id FROM entitlement_grants WHERE member_id = $1))
		   OR (o.resource_type = 'member_device' AND o.resource_id IN (SELECT id FROM member_devices WHERE member_id = $1))
		ORDER BY o.created_at ASC, o.id ASC`
	rows, err := q.Query(ctx, sqlQuery, memberID)
	if err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	out := make([]api.Operation, 0, len(ids))
	for _, id := range ids {
		op, err := scanOperation(ctx, q, id)
		if err != nil {
			return nil, err
		}
		out = append(out, op)
	}
	return out, nil
}

// GetMemberOperation reads an operation by ID only if it belongs to memberID
// (via resource_type = 'member', resource_type = 'entitlement_grant' through entitlement_grants.member_id,
// or resource_type = 'member_device' through member_devices.member_id).
func (s *Service) GetMemberOperation(ctx context.Context, memberID uuid.UUID, opID uuid.UUID) (api.Operation, error) {
	return GetMemberOperation(ctx, s.pool, memberID, opID)
}

// GetMemberOperation reads an operation by ID verifying ownership by memberID across
// member, entitlement_grant (via entitlement_grants.member_id), and member_device (via member_devices.member_id).
func GetMemberOperation(ctx context.Context, q Querier, memberID uuid.UUID, opID uuid.UUID) (api.Operation, error) {
	var matchedID uuid.UUID
	err := q.QueryRow(ctx, `
		SELECT o.id
		FROM operations o
		WHERE o.id = $1
		  AND (
			(o.resource_type = 'member' AND o.resource_id = $2)
			OR (o.resource_type = 'entitlement_grant' AND o.resource_id IN (SELECT id FROM entitlement_grants WHERE member_id = $2))
			OR (o.resource_type = 'member_device' AND o.resource_id IN (SELECT id FROM member_devices WHERE member_id = $2))
		  )`, opID, memberID).Scan(&matchedID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.Operation{}, apierr.Newf(apierr.ResourceNotFound, "operation %s not found", opID)
		}
		return api.Operation{}, err
	}
	return scanOperation(ctx, q, matchedID)
}

func scanOperation(ctx context.Context, q Querier, opID uuid.UUID) (api.Operation, error) {
	const sqlQuery = `
		SELECT
			id, kind, resource_type, coalesce(resource_id, '00000000-0000-0000-0000-000000000000'::uuid),
			coalesce(request_id, id), desired_generation::text, observed_generation::text,
			status, steps, observed_refs, error_code, error_message,
			resource_version::text, created_at, updated_at, completed_at
		FROM operations
		WHERE id = $1`
	var (
		op          api.Operation
		kind        string
		resType     string
		resID       uuid.UUID
		status      string
		desiredGen  *string
		observedGen *string
		stepsBytes  []byte
		refsBytes   []byte
		errCode     *string
		errMsg      *string
		resVer      *string
		completedAt *time.Time
	)
	err := q.QueryRow(ctx, sqlQuery, opID).Scan(
		&op.OperationId, &kind, &resType, &resID,
		&op.RequestId, &desiredGen, &observedGen,
		&status, &stepsBytes, &refsBytes, &errCode, &errMsg,
		&resVer, &op.CreatedAt, &op.UpdatedAt, &completedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.Operation{}, apierr.Newf(apierr.ResourceNotFound, "operation %s not found", opID)
		}
		return api.Operation{}, err
	}
	// §14.1: pgx may decode timestamptz in the process's local timezone.
	op.CreatedAt = op.CreatedAt.UTC()
	op.UpdatedAt = op.UpdatedAt.UTC()
	if completedAt != nil {
		utc := completedAt.UTC()
		completedAt = &utc
	}
	op.CompletedAt = presence.FromPointer(completedAt)
	op.ResourceVersion.SetNull()
	op.DesiredGeneration.SetNull()
	op.ObservedGeneration.SetNull()
	op.Error.SetNull()
	op.Kind = api.OperationKind(kind)
	op.Resource = api.ResourceRef{
		Id:   resID,
		Type: api.ResourceType(resType),
	}
	op.Status = api.OperationStatus(status)
	if resVer != nil {
		if v, err := decstr.Parse(*resVer); err == nil {
			op.ResourceVersion.Set(v)
		}
	}
	if desiredGen != nil {
		if v, err := decstr.Parse(*desiredGen); err == nil {
			op.DesiredGeneration.Set(v)
		}
	}
	if observedGen != nil {
		if v, err := decstr.Parse(*observedGen); err == nil {
			op.ObservedGeneration.Set(v)
		}
	}
	if errCode != nil && *errCode != "" {
		msg := ""
		if errMsg != nil {
			msg = *errMsg
		}
		op.Error.Set(struct {
			Code    api.ErrorCode `json:"code"`
			Message string        `json:"message"`
		}{
			Code:    api.ErrorCode(*errCode),
			Message: msg,
		})
	}

	var storedSteps map[string]StoredStep
	if len(stepsBytes) > 0 {
		_ = json.Unmarshal(stepsBytes, &storedSteps)
	}
	refs := map[string]string{}
	if len(refsBytes) > 0 {
		var rawRefs map[string]json.RawMessage
		if json.Unmarshal(refsBytes, &rawRefs) == nil {
			for k, raw := range rawRefs {
				var s string
				if json.Unmarshal(raw, &s) == nil {
					refs[k] = s
					continue
				}
				var ev struct {
					Reference string `json:"reference"`
				}
				if json.Unmarshal(raw, &ev) == nil && ev.Reference != "" {
					refs[k] = ev.Reference
				}
			}
		}
	}

	type namedStep struct {
		name string
		step StoredStep
	}
	ordered := make([]namedStep, 0, len(storedSteps))
	for k, v := range storedSteps {
		ordered = append(ordered, namedStep{name: k, step: v})
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].step.Order != ordered[j].step.Order {
			return ordered[i].step.Order < ordered[j].step.Order
		}
		return ordered[i].name < ordered[j].name
	})

	apiSteps := make([]api.OperationStep, 0, len(ordered))
	for _, item := range ordered {
		obsAt := presence.NewNullNullable[time.Time]()
		if item.step.ObservedAt != nil && !item.step.ObservedAt.IsZero() {
			obsAt = presence.NewNullableWithValue(item.step.ObservedAt.UTC())
		}
		// §5.2: creating or updating a pending step is not observation evidence.
		// Preserve null until a worker records observed_at, as the core reader does.
		stepOut := api.OperationStep{
			Name:       item.name,
			Required:   item.step.Required,
			Status:     api.OperationStepStatus(item.step.Status),
			ObservedAt: obsAt,
			ErrorCode:  presence.NewNullNullable[api.ErrorCode](),
			Message:    presence.NewNullNullable[string](),
		}
		if item.step.ErrorCode != nil && *item.step.ErrorCode != "" {
			code := api.ErrorCode(*item.step.ErrorCode)
			stepOut.ErrorCode.Set(code)
		}
		if item.step.ErrorMessage != nil && *item.step.ErrorMessage != "" {
			msgCopy := *item.step.ErrorMessage
			stepOut.Message.Set(msgCopy)
		} else if item.step.Reason != "" {
			reasonCopy := item.step.Reason
			stepOut.Message.Set(reasonCopy)
		} else if item.step.ObservedRef != "" {
			if val, ok := refs[item.step.ObservedRef]; ok && val != "" {
				stepOut.Message.Set(val)
			} else {
				refCopy := item.step.ObservedRef
				stepOut.Message.Set(refCopy)
			}
		}
		apiSteps = append(apiSteps, stepOut)
	}
	op.Steps = apiSteps
	return op, nil
}
