package policies

import (
	"context"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// feedStatusTx reads only observations for the selected active revision. Drafts
// and superseded revisions cannot supply either its approval pin or last-good.
func feedStatusTx(ctx context.Context, tx pgx.Tx, revision uuid.UUID, now time.Time) (api.ThreatFeedStatus, error) {
	var approved string
	var good, reason *string
	var count *int
	var goodAt, attemptAt, dueAt *time.Time
	var failures int64
	err := tx.QueryRow(ctx, `SELECT r.payload->>'content_sha256', s.last_good_content,
 c.rule_count, s.last_success_at, s.last_attempt_at, s.last_error_code,
 coalesce(s.consecutive_failures,0), s.next_attempt_at
 FROM policy_revisions r
 LEFT JOIN dns_feed_refresh_state s ON s.feed_revision_id=r.id
 LEFT JOIN dns_feed_contents c ON c.feed_revision_id=s.feed_revision_id AND c.content_sha256=s.last_good_content
 WHERE r.id=$1 AND r.type='threat_feed'`, revision).Scan(
		&approved, &good, &count, &goodAt, &attemptAt, &reason, &failures, &dueAt)
	if err != nil {
		return api.ThreatFeedStatus{}, err
	}
	if good != nil && (count == nil || goodAt == nil || *count < 0 || *count > 500000) {
		return api.ThreatFeedStatus{}, errors.New("invalid feed refresh observation")
	}
	status := api.ThreatFeedStatus{
		ApprovedContentSha256: approved,
		LastGoodContentSha256: presence.FromPointer(good),
		LastGoodRuleCount:     presence.FromPointer(count),
		LastGoodAt:            presence.NewNullNullable[time.Time](),
		LastGoodAgeSeconds:    presence.NewNullNullable[decstr.U64](),
		LastAttemptAt:         utcNullable(attemptAt),
		LastResult:            api.ThreatFeedStatusLastResultNever,
		ConsecutiveFailures:   int(min(max(failures, 0), canonjson.MaxSafeInteger)),
		LastErrorCode:         presence.NewNullNullable[api.ErrorCode](),
		LastErrorMessage:      presence.NewNullNullable[string](),
		NextDueAt:             utcNullable(dueAt),
	}
	if good != nil {
		status.LastGoodAt = utcNullable(goodAt)
		seconds := now.Unix() - goodAt.Unix()
		if now.Nanosecond() < goodAt.Nanosecond() {
			seconds--
		}
		status.LastGoodAgeSeconds.Set(decstr.U64(max(seconds, 0)))
		status.LastResult = api.ThreatFeedStatusLastResultSuccess
	}
	// An attempt in progress updates last_attempt_at only. Keep the result of
	// the last completed attempt (or never), rather than inventing success.
	if reason != nil {
		status.LastResult = api.ThreatFeedStatusLastResultFailed
		status.LastErrorCode.Set(feedPublicErrorCode(*reason))
		message := "Feed refresh failed; retaining last-good content."
		if good == nil {
			message = "Feed refresh failed; no last-good content is available."
		}
		status.LastErrorMessage.Set(message)
	}
	return status, nil
}

func utcNullable(value *time.Time) presence.Nullable[time.Time] {
	if value == nil {
		return presence.NewNullNullable[time.Time]()
	}
	return presence.NewNullableWithValue(value.UTC())
}

// Only these constants cross the public boundary. Unknown stored reasons never
// become an ErrorCode or message; this also bounds/redacts future worker errors.
func feedPublicErrorCode(reason string) api.ErrorCode {
	switch reason {
	case "fetch_failed", "http_status", "read_failed":
		return api.ErrorCodeDEPENDENCYUNAVAILABLE
	case "content_mismatch":
		return api.ErrorCodeHASHMISMATCH
	case "oversize", "content_too_large":
		return api.ErrorCodePAYLOADTOOLARGE
	case "invalid_config", "invalid_size", "outside_validity", "ssrf_refused", "parse_error", "zero_rules", "delta_breach":
		return api.ErrorCodeVALIDATIONFAILED
	default:
		return api.ErrorCodeINTERNALERROR
	}
}
