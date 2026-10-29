package nodes

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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
	Order        int       `json:"order"`
	Required     bool      `json:"required"`
	Status       string    `json:"status"`
	ObservedRef  string    `json:"observed_ref,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	ErrorCode    *string   `json:"error_code,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
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
		p.ResourceType = "node"
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
		resVer      string
	)
	err := q.QueryRow(ctx, sqlQuery, opID).Scan(
		&op.OperationId, &kind, &resType, &resID,
		&op.RequestId, &desiredGen, &observedGen,
		&status, &stepsBytes, &refsBytes, &errCode, &errMsg,
		&resVer, &op.CreatedAt, &op.UpdatedAt, &op.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.Operation{}, apierr.Newf(apierr.ResourceNotFound, "operation %s not found", opID)
		}
		return api.Operation{}, err
	}
	op.Kind = api.OperationKind(kind)
	op.Resource = api.ResourceRef{
		Id:   resID,
		Type: api.ResourceType(resType),
	}
	op.Status = api.OperationStatus(status)
	if v, err := decstr.Parse(resVer); err == nil {
		op.ResourceVersion = &v
	}
	if desiredGen != nil {
		if v, err := decstr.Parse(*desiredGen); err == nil {
			op.DesiredGeneration = &v
		}
	}
	if observedGen != nil {
		if v, err := decstr.Parse(*observedGen); err == nil {
			op.ObservedGeneration = &v
		}
	}
	if errCode != nil && *errCode != "" {
		msg := ""
		if errMsg != nil {
			msg = *errMsg
		}
		op.Error = &struct {
			Code    api.ErrorCode `json:"code"`
			Message string        `json:"message"`
		}{
			Code:    api.ErrorCode(*errCode),
			Message: msg,
		}
	}

	var storedSteps map[string]StoredStep
	if len(stepsBytes) > 0 {
		_ = json.Unmarshal(stepsBytes, &storedSteps)
	}
	var refs map[string]string
	if len(refsBytes) > 0 {
		_ = json.Unmarshal(refsBytes, &refs)
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
		updatedAt := item.step.UpdatedAt
		stepOut := api.OperationStep{
			Name:       item.name,
			Required:   item.step.Required,
			Status:     api.OperationStepStatus(item.step.Status),
			ObservedAt: &updatedAt,
		}
		if item.step.ErrorCode != nil && *item.step.ErrorCode != "" {
			code := api.ErrorCode(*item.step.ErrorCode)
			stepOut.ErrorCode = &code
		}
		if item.step.ErrorMessage != nil && *item.step.ErrorMessage != "" {
			msgCopy := *item.step.ErrorMessage
			stepOut.Message = &msgCopy
		} else if item.step.Reason != "" {
			reasonCopy := item.step.Reason
			stepOut.Message = &reasonCopy
		} else if item.step.ObservedRef != "" {
			if val, ok := refs[item.step.ObservedRef]; ok && val != "" {
				stepOut.Message = &val
			} else {
				refCopy := item.step.ObservedRef
				stepOut.Message = &refCopy
			}
		}
		apiSteps = append(apiSteps, stepOut)
	}
	op.Steps = apiSteps
	return op, nil
}
