package fleetupdate

import (
	"context"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/jackc/pgx/v5"
)

// D-062(d): an authenticated, job-bound result receipt can contain a rejection
// instead of an ACK. It never becomes accepted job evidence or contributes to
// target counts. Keep only safe API fields, never error causes or SQL.
type resultRejection struct {
	Error *struct {
		Code      apierr.Code    `json:"code"`
		Message   string         `json:"message"`
		Retryable bool           `json:"retryable"`
		Details   map[string]any `json:"details,omitempty"`
	} `json:"error,omitempty"`
}

func (r resultRejection) apiError() *apierr.Error {
	return &apierr.Error{Code: r.Error.Code, Message: r.Error.Message, Retryable: &r.Error.Retryable, Details: r.Error.Details}
}
func resultError(err error) error {
	var e *apierr.Error
	if !errors.As(err, &e) {
		return err
	}
	switch e.HTTPStatus() {
	case 400, 401, 403, 404, 422:
		return e.WithRetryable(false)
	case 409:
		return e.WithRetryable(true)
	case 429:
		e = e.WithRetryable(true)
		if e.RetryAfter <= 0 {
			e = e.WithRetryAfter(time.Second)
		}
		return e
	}
	return err
}
func recordResultRejection(ctx context.Context, tx pgx.Tx, i Identity, r api.UpdateResult, raw, digest []byte, err error) error {
	var e *apierr.Error
	if !errors.As(resultError(err), &e) {
		return err
	}
	receipt := map[string]any{"error": map[string]any{"code": e.Code, "message": e.Message, "retryable": e.IsRetryable(), "details": e.Details}}
	response, marshalErr := canonjson.Marshal(receipt)
	if marshalErr != nil {
		return marshalErr
	}
	_, err = tx.Exec(ctx, `INSERT INTO update_results(result_id,job_id,node_id,incarnation_id,payload_hash,payload,ack) VALUES($1,$2,$3,$4,$5,$6,$7)`, r.ResultId, r.JobId, i.NodeID, i.IncarnationID, digest, raw, response)
	return err
}
