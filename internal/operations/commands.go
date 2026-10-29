package operations

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"regexp"
	"slices"
	"strings"
	"time"
)

const failureMessage = "Required operation step failed; quote the operation ID."

// Create must share the domain transaction with desired state, audit and outbox.
func Create(ctx context.Context, tx pgx.Tx, r CreateRequest) (uuid.UUID, error) {
	if tx == nil {
		return uuid.Nil, errors.New("operation creation requires a transaction")
	}
	id := uuid.Must(uuid.NewV7())
	validPrincipal := ((r.Principal.Role == auth.Admin || r.Principal.Role == auth.Member) && r.Principal.ID != uuid.Nil) || (r.Principal.Role == auth.Agent && r.Principal.NodeID != uuid.Nil && r.Principal.IncarnationID != uuid.Nil)
	if len(r.Steps) < 1 || len(r.Steps) > 64 || r.Resource.Id == uuid.Nil || r.RequestID == uuid.Nil || !validPrincipal {
		return uuid.Nil, errors.New("invalid operation")
	}
	if !r.Kind.Valid() || !r.Resource.Type.Valid() {
		return uuid.Nil, errors.New("invalid operation kind or resource type")
	}
	steps := make(map[string]Step, len(r.Steps))
	required := 0
	for name, s := range r.Steps {
		if !regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`).MatchString(name) {
			return uuid.Nil, errors.New("invalid step name")
		}
		if s.Required {
			required++
			s = Step{Required: true, Status: "pending"}
		} else {
			if strings.TrimSpace(s.Reason) == "" || len(s.Reason) > 1000 {
				return uuid.Nil, errors.New("not-required step needs a reason")
			}
			s = Step{Status: "not_required", Reason: s.Reason}
		}
		steps[name] = s
	}
	// §14.5 / DB F-10: new work starts pending and cannot succeed vacuously.
	if required == 0 {
		return uuid.Nil, errors.New("operation needs at least one required step")
	}
	body, _ := json.Marshal(steps)
	_, e := tx.Exec(ctx, `INSERT INTO operations(id,kind,principal,resource_type,resource_id,request_id,resource_version,desired_generation,steps,observed_refs,status) VALUES($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,'{}','pending')`, id, r.Kind, r.Principal.Scope(), r.Resource.Type, r.Resource.Id, r.RequestID, decimal(r.ResourceVersion), decimal(r.DesiredGeneration), body)
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
	var topCode *api.ErrorCode
	var topMessage *string
	// Persist the first failure deterministically. Retry clears obsolete errors.
	names := make([]string, 0, len(steps))
	for name := range steps {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		step := steps[name]
		if step.Status == "failed" && step.ErrorCode != nil {
			topCode = step.ErrorCode
			message := failureMessage
			topMessage = &message
			break
		}
	}
	raw, _ = json.Marshal(steps)
	refsRaw, _ = json.Marshal(refs)
	_, err = tx.Exec(ctx, `UPDATE operations SET steps=$2,observed_refs=$3,status=$4,completed_at=$5,updated_at=clock_timestamp(),observed_generation=coalesce($6::numeric,observed_generation),error_code=$7,error_message=$8 WHERE id=$1`, id, raw, refsRaw, next, completed, decimal(observed), topCode, topMessage)
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
