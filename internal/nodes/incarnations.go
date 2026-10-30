package nodes

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// MaxPaginationCursorLen is the contract bound on pagination cursor strings (§14.1).
const MaxPaginationCursorLen = 512

// ListNodeIncarnations returns the newest-first cursor-paginated history of all
// incarnations recorded for nodeID, together with per-incarnation retained usage
// totals from the usage ledger (`GET /admin/nodes/{node_id}/incarnations`, §6.2,
// §10.4, §14.2, ENR-03).
//
// Read-model sources:
//   - Identity & lifecycle: `nodes` (`id`, `lifecycle_state`, `current_incarnation_id`),
//     `node_incarnations` (`id`, `node_id`, `public_key_fingerprint`, `created_at`, `revoked_at`),
//     `node_enrollments` (`id`, `node_id`, `incarnation_id`, `consumed_at`).
//   - Per-incarnation usage ledger (migration 00130, read-only):
//     `usage_checkpoints` (`node_id`, `incarnation_id`, `updated_at`),
//     `usage_events` (`id`, `node_id`, `incarnation_id`),
//     `usage_samples` (`event_id`, `upload_bytes`, `download_bytes`).
//     Rejected/conflicted events and rollups (`usage_event_rejections`,
//     `usage_event_conflicts`, `usage_rollups`, `usage_rollup_details`) are never read.
func (s *Service) ListNodeIncarnations(
	ctx context.Context,
	nodeID uuid.UUID,
	p api.ListNodeIncarnationsParams,
) (api.NodeIncarnationList, error) {
	limit := DefaultPageLimit
	if p.Limit != nil {
		limit = *p.Limit
		if limit < 1 || limit > MaxPageLimit {
			return api.NodeIncarnationList{}, apierr.Newf(apierr.ValidationFailed, "limit %d must be between 1 and %d", limit, MaxPageLimit)
		}
	}

	var (
		hasCursor       bool
		cursorCreatedAt time.Time
		cursorID        uuid.UUID
	)
	if p.Cursor != nil && *p.Cursor != "" {
		if len(*p.Cursor) > MaxPaginationCursorLen {
			return api.NodeIncarnationList{}, apierr.New(apierr.ValidationFailed, "cursor exceeds maximum length")
		}
		cur, err := decodeNodeCursor(*p.Cursor)
		if err != nil {
			return api.NodeIncarnationList{}, err
		}
		hasCursor = true
		cursorCreatedAt = cur.CreatedAt
		cursorID = cur.ID
	}

	const sqlQuery = `
		SELECT
			n.lifecycle_state,
			n.current_incarnation_id,
			inc.id,
			inc.enrollment_id,
			inc.public_key_fingerprint,
			inc.created_at,
			inc.bound_at,
			inc.revoked_at,
			inc.has_later_incarnation,
			inc.upload_bytes,
			inc.download_bytes,
			inc.through
		FROM nodes n
		LEFT JOIN LATERAL (
			SELECT
				i.id,
				e.id AS enrollment_id,
				i.public_key_fingerprint,
				i.created_at,
				coalesce(e.consumed_at, i.created_at) AS bound_at,
				i.revoked_at,
				EXISTS (
					SELECT 1
					FROM node_incarnations i2
					WHERE i2.node_id = i.node_id
					  AND i2.id <> i.id
					  AND (
						  i2.created_at > i.created_at
						  OR (i2.created_at = i.created_at AND (i2.revoked_at IS NULL OR i2.revoked_at > i.revoked_at OR i2.id > i.id))
					  )
				) AS has_later_incarnation,
				coalesce(sum(s.upload_bytes), 0)::text AS upload_bytes,
				coalesce(sum(s.download_bytes), 0)::text AS download_bytes,
				c.updated_at AS through
			FROM node_incarnations i
			JOIN node_enrollments e ON e.incarnation_id = i.id AND e.node_id = i.node_id
			LEFT JOIN usage_checkpoints c ON c.node_id = i.node_id AND c.incarnation_id = i.id
			LEFT JOIN usage_events ev ON ev.node_id = i.node_id AND ev.incarnation_id = i.id
			LEFT JOIN usage_samples s ON s.event_id = ev.id
			WHERE i.node_id = n.id
			  AND (NOT $2::boolean OR (i.created_at, i.id) < ($3::timestamptz, $4::uuid))
			GROUP BY
				i.id,
				i.node_id,
				i.created_at,
				i.revoked_at,
				i.public_key_fingerprint,
				e.id,
				e.consumed_at,
				c.updated_at
			ORDER BY i.created_at DESC, i.id DESC
			LIMIT $5
		) inc ON true
		WHERE n.id = $1
		ORDER BY inc.created_at DESC, inc.id DESC`

	rows, err := s.pool.Query(ctx, sqlQuery, nodeID, hasCursor, cursorCreatedAt, cursorID, limit+1)
	if err != nil {
		return api.NodeIncarnationList{}, err
	}
	defer rows.Close()

	var (
		sawNode bool
		items   = []api.NodeIncarnation{}
	)
	for rows.Next() {
		var (
			lifecycleState       string
			currentIncarnationID *uuid.UUID
			incID                *uuid.UUID
			enrollmentID         *uuid.UUID
			pubKeyFP             []byte
			createdAt            *time.Time
			boundAt              *time.Time
			revokedAt            *time.Time
			hasLater             *bool
			uploadBytesStr       *string
			downloadBytesStr     *string
			throughAt            *time.Time
		)
		if err := rows.Scan(
			&lifecycleState,
			&currentIncarnationID,
			&incID,
			&enrollmentID,
			&pubKeyFP,
			&createdAt,
			&boundAt,
			&revokedAt,
			&hasLater,
			&uploadBytesStr,
			&downloadBytesStr,
			&throughAt,
		); err != nil {
			return api.NodeIncarnationList{}, err
		}
		sawNode = true
		if incID == nil {
			continue
		}

		upStr := "0"
		if uploadBytesStr != nil {
			upStr = *uploadBytesStr
		}
		upBytes, err := decstr.Parse(upStr)
		if err != nil {
			return api.NodeIncarnationList{}, fmt.Errorf("nodes: incarnation %s upload_bytes sum %q overflows uint64: %w", *incID, upStr, errors.Join(decstr.ErrOverflow, err))
		}

		downStr := "0"
		if downloadBytesStr != nil {
			downStr = *downloadBytesStr
		}
		downBytes, err := decstr.Parse(downStr)
		if err != nil {
			return api.NodeIncarnationList{}, fmt.Errorf("nodes: incarnation %s download_bytes sum %q overflows uint64: %w", *incID, downStr, errors.Join(decstr.ErrOverflow, err))
		}

		through := presence.NewNullNullable[time.Time]()
		if throughAt != nil {
			through.Set(throughAt.UTC())
		}

		state := api.NodeIncarnationStateActive
		retireReason := presence.NewNullNullable[api.NodeIncarnationRetireReason]()
		retiredAt := presence.NewNullNullable[time.Time]()
		if revokedAt != nil {
			state = api.NodeIncarnationStateRetired
			retiredAt.Set(revokedAt.UTC())
			switch {
			case hasLater != nil && *hasLater:
				retireReason.Set(api.NodeIncarnationRetireReasonReinstall)
			case lifecycleState == string(api.LifecycleStateRevoked) || lifecycleState == string(api.LifecycleStateRetired):
				retireReason.Set(api.NodeIncarnationRetireReasonRevocation)
			default:
				retireReason.Set(api.NodeIncarnationRetireReasonSupersession)
			}
		}

		var enrollUUID uuid.UUID
		if enrollmentID != nil {
			enrollUUID = *enrollmentID
		}
		var createdUTC, boundUTC time.Time
		if createdAt != nil {
			createdUTC = createdAt.UTC()
		}
		if boundAt != nil {
			boundUTC = boundAt.UTC()
		}

		items = append(items, api.NodeIncarnation{
			IncarnationId:        *incID,
			EnrollmentId:         enrollUUID,
			BoundPublicKeySha256: hex.EncodeToString(pubKeyFP),
			CreatedAt:            createdUTC,
			BoundAt:              boundUTC,
			Current:              currentIncarnationID != nil && *currentIncarnationID == *incID,
			State:                state,
			RetireReason:         retireReason,
			RetiredAt:            retiredAt,
			Usage: api.NodeIncarnationUsage{
				UploadBytes:   upBytes,
				DownloadBytes: downBytes,
				Through:       through,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return api.NodeIncarnationList{}, err
	}
	if !sawNode {
		return api.NodeIncarnationList{}, apierr.Newf(apierr.ResourceNotFound, "node %s not found", nodeID)
	}

	var nextCursor *string
	if len(items) > limit {
		last := items[limit-1]
		cur := encodeNodeCursor(nodeCursor{CreatedAt: last.CreatedAt, ID: last.IncarnationId})
		nextCursor = &cur
		items = items[:limit]
	}

	return api.NodeIncarnationList{
		Items:      items,
		NextCursor: presence.FromPointer(nextCursor),
	}, nil
}
