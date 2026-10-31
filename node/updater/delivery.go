package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// DeliveryError preserves the HTTP evidence without retaining a response body,
// message, URL or credentials. OpError remains available via errors.As.
type DeliveryError struct {
	HTTPStatus int
	APICode    string
	RetryAt    time.Time
	Op         *OpError
}

func (e *DeliveryError) Error() string {
	return fmt.Sprintf("updater: result HTTP %d API %s: %s", e.HTTPStatus, e.APICode, e.Op.Code)
}

func (e *DeliveryError) Unwrap() error { return e.Op }

func (e *DeliveryError) rejected() bool {
	return e.HTTPStatus >= 400 && e.HTTPStatus < 500 && e.HTTPStatus != 409 && e.HTTPStatus != 429
}

func installerResultPending(payload []byte, status int, code string) bool {
	if status != http.StatusNotFound || code != string(apierr.ResourceNotFound) {
		return false
	}
	var result struct {
		Source api.UpdateResultSource `json:"source"`
	}
	return json.Unmarshal(payload, &result) == nil && result.Source == api.UpdateResultSourceInstaller
}

func responseCode(body []byte) string {
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) != nil || !api.ErrorCode(envelope.Error.Code).Valid() {
		return ""
	}
	return envelope.Error.Code
}

// responseReason retains only D-110's fixed reason, never arbitrary server text.
func responseReason(body []byte) string {
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Details struct {
				Reason string `json:"reason"`
			} `json:"details"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.Error.Code == string(apierr.StateConflict) && envelope.Error.Details.Reason == "job_id_mismatch" {
		return "job_id_mismatch"
	}
	return ""
}

func retryAfter(header string, now time.Time) time.Time {
	header = strings.TrimSpace(header)
	if len(header) <= 128 {
		if seconds, err := strconv.ParseUint(header, 10, 64); err == nil {
			const maxSeconds = uint64((1<<63 - 1) / int64(time.Second))
			if seconds > maxSeconds {
				return time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
			}
			return now.Add(time.Duration(seconds) * time.Second)
		}
		if at, err := http.ParseTime(header); err == nil {
			if at.Before(now) {
				return now
			}
			return at
		}
	}
	// Missing/malformed Retry-After still backs off, rather than spinning.
	return now.Add(time.Second)
}
