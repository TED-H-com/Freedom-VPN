package nodes

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Default and maximum pagination limits (§14.1).
const (
	DefaultPageLimit = 50
	MaxPageLimit     = 100
)

// Outbox event types emitted by internal/nodes.
const (
	EventNodeUpdated = "node.updated"
	EventNodeRevoked = "node.revoked"
	EventNodeRetired = "node.retired"
)

// Service manages the central node registry, aggregate views, mutations,
// observations, and staleness detection (§5, §14.2, §14.4, §15.3).
type Service struct {
	pool           *pgxpool.Pool
	now            func() time.Time
	observeVersion func(context.Context, pgx.Tx, api.ObservationReport) error
}

// Option configures a Service.
type Option func(*Service)

// WithVersionObserver joins deduplicated observation receipts and fleet evidence (§13.8).
func WithVersionObserver(fn func(context.Context, pgx.Tx, api.ObservationReport) error) Option {
	return func(s *Service) { s.observeVersion = fn }
}

// WithClock overrides the clock used by Service (for deterministic tests).
func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

// NewService constructs a node registry Service backed by pool.
func NewService(pool *pgxpool.Pool, opts ...Option) *Service {
	s := &Service{
		pool: pool,
		now:  func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// FormatETag formats a decimal resource_version string as a quoted HTTP ETag (§14.5).
func FormatETag(resourceVersion string) string {
	return `"` + strings.Trim(resourceVersion, `"`) + `"`
}

// ParseIfMatch extracts the unquoted decimal resource_version from an If-Match header.
// Returns a 428 PRECONDITION_REQUIRED error if the header is missing or empty,
// or a 412 RESOURCE_VERSION_CONFLICT error if it is malformed (§14.5).
func ParseIfMatch(ifMatch string) (string, error) {
	trimmed := strings.TrimSpace(ifMatch)
	if trimmed == "" {
		return "", apierr.New(apierr.PreconditionRequired, "If-Match header is required for this mutation")
	}
	if len(trimmed) < 3 || trimmed[0] != '"' || trimmed[len(trimmed)-1] != '"' {
		return "", apierr.New(apierr.ResourceVersionConflict, "Strong If-Match required")
	}
	unquoted := trimmed[1 : len(trimmed)-1]
	if u, err := decstr.Parse(unquoted); err != nil || uint64(u) == 0 {
		return "", apierr.New(apierr.ResourceVersionConflict, "If-Match header does not match a valid resource_version")
	}
	return unquoted, nil
}

// MapDBPublicationObserved maps the database exits.publication_observed value
// (`inactive`, `applying`, `active`, `drifted`, `error`) onto api.PublicationObserved.
func MapDBPublicationObserved(dbVal string) api.PublicationObserved {
	switch dbVal {
	case "active":
		return api.PublicationObservedActive
	case "applying":
		return api.PublicationObservedApplying
	case "error", "drifted":
		return api.PublicationObservedError
	case "inactive", "withdrawn":
		return api.PublicationObservedWithdrawn
	default:
		return api.PublicationObservedAbsent
	}
}

// GetNode returns the full admin node detail view and its HTTP ETag (§5.1, §14.2, §15.3).
// publication_intent and publication_observed are joined from exits (§15.3).
func (s *Service) GetNode(ctx context.Context, nodeID uuid.UUID) (api.NodeDetail, string, error) {
	return getNodeWithQuerier(ctx, s.pool, nodeID)
}

// GetNodeTx returns the full admin node detail view inside an active transaction.
func GetNodeTx(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID) (api.NodeDetail, string, error) {
	return getNodeWithQuerier(ctx, tx, nodeID)
}

func getNodeWithQuerier(ctx context.Context, q Querier, nodeID uuid.UUID) (api.NodeDetail, string, error) {
	const sqlQuery = `
		SELECT
			n.id,
			e.id AS exit_id,
			e.display_name,
			e.region,
			n.lifecycle_state,
			n.provisioning_state,
			coalesce(e.publication_intent, 'unpublished'),
			coalesce(e.publication_observed, 'absent'),
			n.control_status,
			n.data_health,
			n.safety_status,
			n.member_forwarding_observed,
			n.current_incarnation_id,
			inc.peer_id,
			coalesce(inc.os, 'ubuntu'),
			coalesce(inc.os_version, '24.04'),
			coalesce(inc.arch, 'amd64'),
			n.desired_config_generation::text,
			n.applied_config_generation::text,
			n.desired_release_id,
			n.current_release_id,
			n.last_good_release_id,
			n.last_seen_at,
			n.notes,
			n.capacity,
			n.observed_exit_ipv4,
			n.observed_exit_ipv6,
			n.latest_checks,
			n.recent_errors,
			n.drift,
			n.resource_version::text,
			n.created_at,
			n.retired_at
		FROM nodes n
		LEFT JOIN exits e ON e.node_id = n.id
		LEFT JOIN node_incarnations inc ON inc.id = n.current_incarnation_id
		WHERE n.id = $1`

	var (
		detail                   api.NodeDetail
		exitID                   *uuid.UUID
		displayName              *string
		region                   *string
		lifecycleState           string
		provisioningState        string
		publicationIntent        string
		publicationObserved      string
		controlStatus            string
		dataHealth               string
		safetyStatus             string
		memberForwardingObserved *bool
		currentIncarnationID     *uuid.UUID
		netbirdPeerID            *string
		osName                   string
		osVersion                string
		arch                     string
		desiredConfigGen         string
		appliedConfigGen         string
		desiredRelease           *uuid.UUID
		currentRelease           *uuid.UUID
		lastGoodRelease          *uuid.UUID
		lastSeenAt               *time.Time
		notes                    string
		capacityBytes            []byte
		observedExitIPv4         *string
		observedExitIPv6         *string
		checksBytes              []byte
		errorsBytes              []byte
		driftBytes               []byte
		resourceVer              string
		createdAt                time.Time
		retiredAt                *time.Time
	)

	err := q.QueryRow(ctx, sqlQuery, nodeID).Scan(
		&detail.NodeId,
		&exitID,
		&displayName,
		&region,
		&lifecycleState,
		&provisioningState,
		&publicationIntent,
		&publicationObserved,
		&controlStatus,
		&dataHealth,
		&safetyStatus,
		&memberForwardingObserved,
		&currentIncarnationID,
		&netbirdPeerID,
		&osName,
		&osVersion,
		&arch,
		&desiredConfigGen,
		&appliedConfigGen,
		&desiredRelease,
		&currentRelease,
		&lastGoodRelease,
		&lastSeenAt,
		&notes,
		&capacityBytes,
		&observedExitIPv4,
		&observedExitIPv6,
		&checksBytes,
		&errorsBytes,
		&driftBytes,
		&resourceVer,
		&createdAt,
		&retiredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NodeDetail{}, "", apierr.Newf(apierr.ResourceNotFound, "node %s not found", nodeID)
		}
		return api.NodeDetail{}, "", err
	}

	detail.Name.SetNull()
	detail.Region.SetNull()
	detail.Notes.SetNull()
	detail.Capacity.SetNull()
	detail.Update, err = updateSummary(ctx, q, nodeID)
	if err != nil {
		return api.NodeDetail{}, "", err
	}
	detail.ExitId = presence.FromPointer(exitID)
	if displayName != nil && *displayName != "" {
		detail.Name = presence.FromPointer(displayName)
	}
	if region != nil && *region != "" {
		detail.Region = presence.FromPointer(region)
	}
	detail.LifecycleState = api.LifecycleState(lifecycleState)
	detail.ProvisioningState = api.ProvisioningState(provisioningState)
	detail.PublicationIntent = api.PublicationIntent(publicationIntent)
	detail.PublicationObserved = MapDBPublicationObserved(publicationObserved)
	detail.ControlStatus = api.ControlStatus(controlStatus)
	detail.DataHealth = api.DataHealth(dataHealth)
	detail.SafetyStatus = api.SafetyStatus(safetyStatus)
	detail.MemberForwardingObserved = presence.FromPointer(memberForwardingObserved)
	detail.CurrentIncarnationId = presence.FromPointer(currentIncarnationID)
	detail.NetbirdPeerId = presence.FromPointer(netbirdPeerID)
	detail.Platform.Set(api.NodePlatform{
		Arch:      api.Architecture(arch),
		Os:        api.NodePlatformOs(osName),
		OsVersion: osVersion,
	})

	cfgGen := api.ConfigGeneration{Applied: presence.NewNullNullable[decstr.U64]()}
	if v, err := decstr.Parse(desiredConfigGen); err == nil {
		cfgGen.Desired = v
	}
	if v, err := decstr.Parse(appliedConfigGen); err == nil {
		cfgGen.Applied.Set(v)
	}
	detail.ConfigGeneration = cfgGen

	detail.Releases = api.NodeReleases{
		DesiredReleaseId:  presence.FromPointer(desiredRelease),
		CurrentReleaseId:  presence.FromPointer(currentRelease),
		LastGoodReleaseId: presence.FromPointer(lastGoodRelease),
	}
	detail.LastSeenAt = presence.FromPointer(lastSeenAt)
	if notes != "" {
		detail.Notes = presence.FromPointer(&notes)
	}
	detail.ObservedExitIpv4 = presence.FromPointer(observedExitIPv4)
	detail.ObservedExitIpv6 = presence.FromPointer(observedExitIPv6)
	if v, err := decstr.Parse(resourceVer); err == nil {
		detail.ResourceVersion = v
	}
	detail.CreatedAt = createdAt
	detail.RetiredAt = presence.FromPointer(retiredAt)

	if len(capacityBytes) > 0 {
		var capObj api.NodeCapacity
		if err := json.Unmarshal(capacityBytes, &capObj); err == nil {
			detail.Capacity = presence.FromPointer(&capObj)
		}
	}

	detail.Checks = []api.NodeCheck{}
	if len(checksBytes) > 0 {
		_ = json.Unmarshal(checksBytes, &detail.Checks)
	}
	detail.RecentErrors = []api.NodeError{}
	if len(errorsBytes) > 0 {
		_ = json.Unmarshal(errorsBytes, &detail.RecentErrors)
	}
	detail.Drift = []api.StateDrift{}
	if len(driftBytes) > 0 {
		_ = json.Unmarshal(driftBytes, &detail.Drift)
	}

	// Load member_group_ids from exit_groups (§15.3).
	groupIDs := []uuid.UUID{}
	if exitID != nil {
		rows, err := q.Query(ctx, `SELECT group_id FROM exit_groups WHERE exit_id = $1 ORDER BY group_id ASC`, *exitID)
		if err != nil {
			return api.NodeDetail{}, "", err
		}
		defer rows.Close()
		for rows.Next() {
			var gid uuid.UUID
			if err := rows.Scan(&gid); err != nil {
				return api.NodeDetail{}, "", err
			}
			groupIDs = append(groupIDs, gid)
		}
		if err := rows.Err(); err != nil {
			return api.NodeDetail{}, "", err
		}
	}
	detail.MemberGroupIds = groupIDs

	return detail, FormatETag(resourceVer), nil
}

// ListNodesParams specifies filtering and cursor pagination options for ListNodes (§14.2).
type ListNodesParams struct {
	Cursor            string
	Limit             int
	LifecycleState    *api.LifecycleState
	PublicationIntent *api.PublicationIntent
	ControlStatus     *api.ControlStatus
	Region            *string
}

type nodeCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

func encodeNodeCursor(c nodeCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeNodeCursor(raw string) (nodeCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nodeCursor{}, apierr.New(apierr.MalformedRequest, "invalid pagination cursor")
	}
	var c nodeCursor
	if err := json.Unmarshal(b, &c); err != nil || c.ID == uuid.Nil || c.CreatedAt.IsZero() {
		return nodeCursor{}, apierr.New(apierr.MalformedRequest, "invalid pagination cursor payload")
	}
	return c, nil
}

// ListNodes returns a filtered, cursor-paginated page of NodeSummary items (§5.1, §14.2).
func (s *Service) ListNodes(ctx context.Context, p ListNodesParams) (api.NodeList, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		return api.NodeList{}, apierr.Newf(apierr.ValidationFailed, "limit %d exceeds maximum %d", limit, MaxPageLimit)
	}

	var (
		clauses []string
		args    []any
	)
	addArg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if p.LifecycleState != nil {
		clauses = append(clauses, "n.lifecycle_state = "+addArg(string(*p.LifecycleState)))
	}
	if p.PublicationIntent != nil {
		clauses = append(clauses, "coalesce(e.publication_intent, 'unpublished') = "+addArg(string(*p.PublicationIntent)))
	}
	if p.ControlStatus != nil {
		clauses = append(clauses, "n.control_status = "+addArg(string(*p.ControlStatus)))
	}
	if p.Region != nil && *p.Region != "" {
		clauses = append(clauses, "e.region = "+addArg(*p.Region))
	}
	if p.Cursor != "" {
		cur, err := decodeNodeCursor(p.Cursor)
		if err != nil {
			return api.NodeList{}, err
		}
		tsArg := addArg(cur.CreatedAt)
		idArg := addArg(cur.ID)
		clauses = append(clauses, fmt.Sprintf("(n.created_at, n.id) < (%s, %s)", tsArg, idArg))
	}

	whereSQL := ""
	if len(clauses) > 0 {
		whereSQL = "WHERE " + strings.Join(clauses, " AND ")
	}
	limitArg := addArg(limit + 1)

	query := fmt.Sprintf(`
		SELECT
			n.id,
			e.id AS exit_id,
			e.display_name,
			e.region,
			n.lifecycle_state,
			n.provisioning_state,
			coalesce(e.publication_intent, 'unpublished'),
			coalesce(e.publication_observed, 'absent'),
			n.control_status,
			n.data_health,
			n.safety_status,
			n.member_forwarding_observed,
			n.desired_config_generation::text,
			n.applied_config_generation::text,
			n.desired_release_id,
			n.current_release_id,
			n.last_good_release_id,
			n.last_seen_at,
			n.resource_version::text,
			n.created_at
		FROM nodes n
		LEFT JOIN exits e ON e.node_id = n.id
		%s
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT %s`, whereSQL, limitArg)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return api.NodeList{}, err
	}
	defer rows.Close()

	items := []api.NodeSummary{}
	exitIDs := []uuid.UUID{}
	exitIndex := map[uuid.UUID][]int{}

	for rows.Next() {
		var (
			item                     api.NodeSummary
			exitID                   *uuid.UUID
			displayName              *string
			region                   *string
			lifecycleState           string
			provisioningState        string
			publicationIntent        string
			publicationObserved      string
			controlStatus            string
			dataHealth               string
			safetyStatus             string
			memberForwardingObserved *bool
			desiredConfigGen         string
			appliedConfigGen         string
			desiredRelease           *uuid.UUID
			currentRelease           *uuid.UUID
			lastGoodRelease          *uuid.UUID
			lastSeenAt               *time.Time
			resourceVer              string
			createdAt                time.Time
		)
		if err := rows.Scan(
			&item.NodeId,
			&exitID,
			&displayName,
			&region,
			&lifecycleState,
			&provisioningState,
			&publicationIntent,
			&publicationObserved,
			&controlStatus,
			&dataHealth,
			&safetyStatus,
			&memberForwardingObserved,
			&desiredConfigGen,
			&appliedConfigGen,
			&desiredRelease,
			&currentRelease,
			&lastGoodRelease,
			&lastSeenAt,
			&resourceVer,
			&createdAt,
		); err != nil {
			return api.NodeList{}, err
		}

		item.Name.SetNull()
		item.Region.SetNull()
		item.Update.SetNull()
		item.ExitId = presence.FromPointer(exitID)
		if displayName != nil && *displayName != "" {
			item.Name = presence.FromPointer(displayName)
		}
		if region != nil && *region != "" {
			item.Region = presence.FromPointer(region)
		}
		item.LifecycleState = api.LifecycleState(lifecycleState)
		item.ProvisioningState = api.ProvisioningState(provisioningState)
		item.PublicationIntent = api.PublicationIntent(publicationIntent)
		item.PublicationObserved = MapDBPublicationObserved(publicationObserved)
		item.ControlStatus = api.ControlStatus(controlStatus)
		item.DataHealth = api.DataHealth(dataHealth)
		item.SafetyStatus = api.SafetyStatus(safetyStatus)
		item.MemberForwardingObserved = presence.FromPointer(memberForwardingObserved)

		cfgGen := api.ConfigGeneration{Applied: presence.NewNullNullable[decstr.U64]()}
		if v, err := decstr.Parse(desiredConfigGen); err == nil {
			cfgGen.Desired = v
		}
		if v, err := decstr.Parse(appliedConfigGen); err == nil {
			cfgGen.Applied.Set(v)
		}
		item.ConfigGeneration = cfgGen

		item.Releases = api.NodeReleases{
			DesiredReleaseId:  presence.FromPointer(desiredRelease),
			CurrentReleaseId:  presence.FromPointer(currentRelease),
			LastGoodReleaseId: presence.FromPointer(lastGoodRelease),
		}
		item.LastSeenAt = presence.FromPointer(lastSeenAt)
		if v, err := decstr.Parse(resourceVer); err == nil {
			item.ResourceVersion = v
		}
		item.CreatedAt = createdAt
		item.MemberGroupIds = []uuid.UUID{}

		idx := len(items)
		items = append(items, item)
		if exitID != nil && idx < limit {
			if _, seen := exitIndex[*exitID]; !seen {
				exitIDs = append(exitIDs, *exitID)
			}
			exitIndex[*exitID] = append(exitIndex[*exitID], idx)
		}
	}
	if err := rows.Err(); err != nil {
		return api.NodeList{}, err
	}
	rows.Close()

	var nextCursor *string
	if len(items) > limit {
		last := items[limit-1]
		cur := encodeNodeCursor(nodeCursor{CreatedAt: last.CreatedAt, ID: last.NodeId})
		nextCursor = &cur
		items = items[:limit]
	}

	if len(exitIDs) > 0 {
		gRows, err := s.pool.Query(ctx, `
			SELECT exit_id, group_id
			FROM exit_groups
			WHERE exit_id = ANY($1)
			ORDER BY group_id ASC`, exitIDs)
		if err != nil {
			return api.NodeList{}, err
		}
		defer gRows.Close()
		for gRows.Next() {
			var eid, gid uuid.UUID
			if err := gRows.Scan(&eid, &gid); err != nil {
				return api.NodeList{}, err
			}
			for _, idx := range exitIndex[eid] {
				items[idx].MemberGroupIds = append(items[idx].MemberGroupIds, gid)
			}
		}
		if err := gRows.Err(); err != nil {
			return api.NodeList{}, err
		}
	}

	for i := range items {
		var e error
		items[i].Update, e = updateSummary(ctx, s.pool, items[i].NodeId)
		if e != nil {
			return api.NodeList{}, e
		}
	}
	return api.NodeList{
		Items:      items,
		NextCursor: presence.FromPointer(nextCursor),
	}, nil
}

// ListMemberExits returns the member-visible exit list (`GET /member/exits`, §14.3)
// for a member entitled to the exits' member groups via D-030 serving eligibility.
func (s *Service) ListMemberExits(ctx context.Context, memberID uuid.UUID) (api.MemberExitList, error) {
	now := s.now().UTC()
	groupIDs, err := entitlement.ServingExitGroups(ctx, s.pool, memberID, now)
	if err != nil {
		return api.MemberExitList{}, err
	}
	if len(groupIDs) == 0 {
		return api.MemberExitList{
			Items:      []api.MemberExit{},
			NextCursor: presence.NewNullNullable[string](),
		}, nil
	}

	const q = `
		SELECT DISTINCT
			e.id,
			e.display_name,
			e.region,
			e.publication_observed,
			n.control_status,
			n.data_health,
			n.safety_status
		FROM exits e
		JOIN nodes n ON n.id = e.node_id
		JOIN exit_groups eg ON eg.exit_id = e.id
		WHERE eg.group_id = ANY($1)
		  AND e.publication_intent = 'published'
		  AND n.lifecycle_state = 'registered'
		ORDER BY e.region ASC, e.display_name ASC, e.id ASC`

	rows, err := s.pool.Query(ctx, q, groupIDs)
	if err != nil {
		return api.MemberExitList{}, err
	}
	defer rows.Close()

	items := []api.MemberExit{}
	for rows.Next() {
		var (
			exitID              uuid.UUID
			name                string
			region              string
			publicationObserved string
			controlStatus       string
			dataHealth          string
			safetyStatus        string
		)
		if err := rows.Scan(&exitID, &name, &region, &publicationObserved, &controlStatus, &dataHealth, &safetyStatus); err != nil {
			return api.MemberExitList{}, err
		}
		availability := api.MemberExitAvailabilityAvailable
		if publicationObserved != "active" ||
			controlStatus != string(api.ControlStatusOnline) ||
			safetyStatus != string(api.SafetyStatusPassed) ||
			dataHealth == string(api.DataHealthFailed) {
			availability = api.MemberExitAvailabilityTemporarilyUnavailable
		}
		items = append(items, api.MemberExit{
			ExitId:       exitID,
			DisplayName:  name,
			Region:       region,
			Availability: availability,
			Note:         presence.NewNullNullable[api.LocalizedText](),
		})
	}
	if err := rows.Err(); err != nil {
		return api.MemberExitList{}, err
	}
	return api.MemberExitList{Items: items,
		NextCursor: presence.NewNullNullable[string]()}, nil
}

// UpdateNodeParams holds inputs for PATCH /admin/nodes/{id} (§14.2, §14.5).
type UpdateNodeParams struct {
	NodeID         uuid.UUID
	Principal      string
	IdempotencyKey uuid.UUID
	IfMatch        string
	RequestID      *uuid.UUID
	Patch          api.NodePatch
}

// UpdateNodeResult holds the outcome of UpdateNode: either a 200 synchronous
// NodeDetail update (metadata-only) or a 202 asynchronous Operation (policy-affecting).
type UpdateNodeResult struct {
	Status    int
	ETag      string
	Node      *api.NodeDetail
	Operation *api.Operation
}

// UpdateNode applies a PATCH to a node using optimistic concurrency (UpdateWithVersion)
// and idempotency (§5.1, §14.2, §14.5). Metadata-only changes return 200 with NodeDetail;
// changes to member_group_ids, capacity, or clear_capacity return 202 with an Operation.
func (s *Service) UpdateNode(ctx context.Context, p UpdateNodeParams) (*UpdateNodeResult, error) {
	if p.IdempotencyKey == uuid.Nil {
		return nil, apierr.New(apierr.MalformedRequest, "Idempotency-Key header is required")
	}
	expectedVer, err := ParseIfMatch(p.IfMatch)
	if err != nil {
		return nil, err
	}
	if p.Patch.ClearCapacity != nil && bool(*p.Patch.ClearCapacity) && p.Patch.Capacity != nil {
		return nil, apierr.Validation("clear_capacity cannot be true when capacity is also provided",
			apierr.FieldError{Field: "/clear_capacity", Reason: "conflicts_with_capacity"},
		)
	}
	if p.Patch.Name != nil && strings.TrimSpace(*p.Patch.Name) == "" {
		return nil, apierr.Validation("name must not be empty",
			apierr.FieldError{Field: "/name", Reason: "required"},
		)
	}
	if p.Patch.Region != nil && strings.TrimSpace(*p.Patch.Region) == "" {
		return nil, apierr.Validation("region must not be empty",
			apierr.FieldError{Field: "/region", Reason: "required"},
		)
	}

	actorUUID, actorType, principal := ResolveAdminActor(ctx, p.Principal)
	reqHash, err := hashIdempotentRequest(p.NodeID.String(), p.Patch)
	if err != nil {
		return nil, err
	}
	keyHash := sha256.Sum256([]byte(p.IdempotencyKey.String()))
	scope := db.IdempotencyScope{
		Principal:   principal,
		Method:      "PATCH",
		Route:       "/api/v1/admin/nodes/" + p.NodeID.String(),
		KeyHash:     keyHash[:],
		RequestHash: reqHash,
	}

	var result *UpdateNodeResult
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		rec, err := commandtx.BeginIdempotent(ctx, tx, scope, 0)
		if err != nil {
			if errors.Is(err, db.ErrIdempotencyConflict) {
				return apierr.New(apierr.IdempotencyConflict, "Idempotency-Key reused with a different request payload")
			}
			return err
		}
		if !rec.New {
			if rec.State == "in_progress" && rec.OperationID == nil {
				return apierr.New(apierr.IdempotencyConflict, "request with this Idempotency-Key is currently in progress")
			}
			detail, etag, err := GetNodeTx(ctx, tx, p.NodeID)
			if err != nil {
				return err
			}
			if rec.OperationID != nil {
				op, err := GetOperationTx(ctx, tx, *rec.OperationID)
				if err != nil {
					return err
				}
				result = &UpdateNodeResult{
					Status:    202,
					ETag:      etag,
					Operation: &op,
				}
				return nil
			}
			result = &UpdateNodeResult{
				Status: 200,
				ETag:   etag,
				Node:   &detail,
			}
			return nil
		}

		beforeDetail, _, err := GetNodeTx(ctx, tx, p.NodeID)
		if err != nil {
			return err
		}
		if beforeDetail.LifecycleState == api.LifecycleStateRetired {
			return apierr.New(apierr.StateConflict, "retired node cannot be modified")
		}

		nodeUpdates := map[string]any{
			"updated_at": s.now(),
		}
		if p.Patch.Notes != nil {
			nodeUpdates["notes"] = *p.Patch.Notes
		}
		if p.Patch.ClearCapacity != nil && bool(*p.Patch.ClearCapacity) {
			nodeUpdates["capacity"] = nil
		} else if p.Patch.Capacity != nil {
			capBytes, err := json.Marshal(p.Patch.Capacity)
			if err != nil {
				return err
			}
			nodeUpdates["capacity"] = capBytes
		}

		newVer, err := db.UpdateWithVersion(ctx, tx, "nodes", p.NodeID, expectedVer, nodeUpdates)
		if err != nil {
			if errors.Is(err, db.ErrVersionConflict) {
				return versionConflictError(ctx, tx, "nodes", p.NodeID)
			}
			return err
		}

		if exitID, err := beforeDetail.ExitId.Get(); err == nil {
			if p.Patch.Name != nil || p.Patch.Region != nil {
				name := ""
				if v, err := beforeDetail.Name.Get(); err == nil {
					name = v
				}
				if p.Patch.Name != nil {
					name = strings.TrimSpace(*p.Patch.Name)
				}
				region := ""
				if v, err := beforeDetail.Region.Get(); err == nil {
					region = v
				}
				if p.Patch.Region != nil {
					region = strings.TrimSpace(*p.Patch.Region)
				}
				if _, err := tx.Exec(ctx, `
					UPDATE exits
					SET display_name = $1, region = $2, resource_version = resource_version + 1
					WHERE id = $3`, name, region, exitID); err != nil {
					return err
				}
			}
			if p.Patch.MemberGroupIds != nil {
				if _, err := tx.Exec(ctx, `DELETE FROM exit_groups WHERE exit_id = $1`, exitID); err != nil {
					return err
				}
				seen := map[uuid.UUID]bool{}
				for _, gid := range *p.Patch.MemberGroupIds {
					if seen[gid] {
						continue
					}
					seen[gid] = true
					if _, err := tx.Exec(ctx, `INSERT INTO exit_groups (exit_id, group_id) VALUES ($1, $2)`, exitID, gid); err != nil {
						var pgErr *pgconn.PgError
						if errors.As(err, &pgErr) && pgErr.Code == "23503" {
							return apierr.Validation(fmt.Sprintf("member_group_id %s does not exist", gid),
								apierr.FieldError{Field: "/member_group_ids", Reason: "not_found"},
							)
						}
						return err
					}
				}
			}
		}

		afterDetail, etag, err := GetNodeTx(ctx, tx, p.NodeID)
		if err != nil {
			return err
		}

		policyAffected := p.Patch.MemberGroupIds != nil ||
			p.Patch.Capacity != nil ||
			(p.Patch.ClearCapacity != nil && bool(*p.Patch.ClearCapacity))

		beforeJSON, _ := json.Marshal(map[string]any{
			"name":             beforeDetail.Name,
			"region":           beforeDetail.Region,
			"notes":            beforeDetail.Notes,
			"member_group_ids": beforeDetail.MemberGroupIds,
			"capacity":         beforeDetail.Capacity,
			"resource_version": beforeDetail.ResourceVersion.String(),
		})
		afterJSON, _ := json.Marshal(map[string]any{
			"name":             afterDetail.Name,
			"region":           afterDetail.Region,
			"notes":            afterDetail.Notes,
			"member_group_ids": afterDetail.MemberGroupIds,
			"capacity":         afterDetail.Capacity,
			"resource_version": afterDetail.ResourceVersion.String(),
		})

		now := s.now()
		if policyAffected {
			nextGen := uint64(afterDetail.ConfigGeneration.Desired) + 1
			op, err := CreateOperation(ctx, tx, CreateOperationParams{
				Kind:              OpKindUpdate,
				Principal:         principal,
				ResourceType:      "node",
				ResourceID:        p.NodeID,
				ResourceVersion:   &newVer,
				RequestID:         p.RequestID,
				DesiredGeneration: &nextGen,
				Now:               now,
			})
			if err != nil {
				return err
			}
			payload, _ := json.Marshal(map[string]any{
				"node_id":          p.NodeID.String(),
				"operation_id":     op.OperationId.String(),
				"resource_version": newVer,
				"occurred_at":      now.Format(time.RFC3339Nano),
			})
			if err := db.EnqueueOutbox(ctx, tx, db.OutboxEvent{
				AggregateID:      p.NodeID,
				AggregateType:    "node",
				AggregateVersion: newVer,
				EventType:        EventNodeUpdated,
				SchemaVersion:    1,
				Payload:          payload,
			}); err != nil {
				return err
			}
			if err := db.AppendAudit(ctx, tx, db.AuditEvent{
				Actor:         actorUUID.String(),
				ActorType:     actorType,
				Action:        "node.update",
				Resource:      "node:" + p.NodeID.String(),
				Result:        "succeeded",
				RequestID:     p.RequestID,
				OperationID:   &op.OperationId,
				SchemaVersion: 1,
				Before:        beforeJSON,
				After:         afterJSON,
			}); err != nil {
				return err
			}
			nodeIDCopy := p.NodeID
			if err := commandtx.CompleteIdempotent(ctx, tx, rec.ID, db.IdempotencyResult{
				OperationID:    &op.OperationId,
				ResourceType:   "node",
				ResourceID:     &nodeIDCopy,
				ResponseStatus: 202,
			}); err != nil {
				return err
			}
			result = &UpdateNodeResult{
				Status:    202,
				ETag:      etag,
				Operation: &op,
			}
			return nil
		}

		if err := db.AppendAudit(ctx, tx, db.AuditEvent{
			Actor:         actorUUID.String(),
			ActorType:     actorType,
			Action:        "node.update",
			Resource:      "node:" + p.NodeID.String(),
			Result:        "succeeded",
			RequestID:     p.RequestID,
			SchemaVersion: 1,
			Before:        beforeJSON,
			After:         afterJSON,
		}); err != nil {
			return err
		}
		nodeIDCopy := p.NodeID
		if err := commandtx.CompleteIdempotent(ctx, tx, rec.ID, db.IdempotencyResult{
			ResourceType:   "node",
			ResourceID:     &nodeIDCopy,
			ResponseStatus: 200,
		}); err != nil {
			return err
		}
		result = &UpdateNodeResult{
			Status: 200,
			ETag:   etag,
			Node:   &afterDetail,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// RevokeNodeParams holds inputs for POST /admin/nodes/{id}/revoke (§14.2).
type RevokeNodeParams struct {
	NodeID         uuid.UUID
	Principal      string
	IdempotencyKey uuid.UUID
	IfMatch        string
	RequestID      *uuid.UUID
	Request        api.ReasonRequest
}

// RevokeNode transitions a node to lifecycle_state=revoked, disables its exit publication,
// revokes active incarnations/certificates/enrollments, and starts a node.revoke operation (§5.1, §6.2, §14.2).
func (s *Service) RevokeNode(ctx context.Context, p RevokeNodeParams) (api.Operation, string, error) {
	return s.revokeOrRetireNode(ctx, p.NodeID, p.Principal, p.IdempotencyKey, p.IfMatch, p.RequestID, p.Request.Reason, false)
}

// RetireNodeParams holds inputs for DELETE /admin/nodes/{id} (§14.2).
type RetireNodeParams struct {
	NodeID         uuid.UUID
	Principal      string
	IdempotencyKey uuid.UUID
	IfMatch        string
	RequestID      *uuid.UUID
	Reason         string
}

// RetireNode transitions a node to the terminal tombstone state lifecycle_state=retired,
// preserves all node, usage ledger, and audit rows (never deleting from nodes),
// disables exit publication, revokes credentials, and returns a node.retire operation (§5.1, §14.2, §15.3).
func (s *Service) RetireNode(ctx context.Context, p RetireNodeParams) (api.Operation, string, error) {
	reason := strings.TrimSpace(p.Reason)
	if reason == "" {
		reason = "node retired by administrator"
	}
	return s.revokeOrRetireNode(ctx, p.NodeID, p.Principal, p.IdempotencyKey, p.IfMatch, p.RequestID, reason, true)
}

func (s *Service) revokeOrRetireNode(
	ctx context.Context,
	nodeID uuid.UUID,
	principal string,
	idempotencyKey uuid.UUID,
	ifMatch string,
	requestID *uuid.UUID,
	reason string,
	retire bool,
) (api.Operation, string, error) {
	if idempotencyKey == uuid.Nil {
		return api.Operation{}, "", apierr.New(apierr.MalformedRequest, "Idempotency-Key header is required")
	}
	expectedVer, err := ParseIfMatch(ifMatch)
	if err != nil {
		return api.Operation{}, "", err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return api.Operation{}, "", apierr.Validation("reason is required",
			apierr.FieldError{Field: "/reason", Reason: "required"},
		)
	}
	actorUUID, actorType, principal := ResolveAdminActor(ctx, principal)

	targetState := api.LifecycleStateRevoked
	method := "POST"
	route := "/api/v1/admin/nodes/" + nodeID.String() + "/revoke"
	opKind := OpKindRevoke
	eventType := EventNodeRevoked
	if retire {
		targetState = api.LifecycleStateRetired
		method = "DELETE"
		route = "/api/v1/admin/nodes/" + nodeID.String()
		opKind = OpKindRetire
		eventType = EventNodeRetired
	}

	reqHash, err := hashIdempotentRequest(nodeID.String(), map[string]any{
		"action": opKind,
		"reason": reason,
	})
	if err != nil {
		return api.Operation{}, "", err
	}
	keyHash := sha256.Sum256([]byte(idempotencyKey.String()))
	scope := db.IdempotencyScope{
		Principal:   principal,
		Method:      method,
		Route:       route,
		KeyHash:     keyHash[:],
		RequestHash: reqHash,
	}

	var (
		outOp   api.Operation
		outETag string
	)
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		rec, err := commandtx.BeginIdempotent(ctx, tx, scope, 0)
		if err != nil {
			if errors.Is(err, db.ErrIdempotencyConflict) {
				return apierr.New(apierr.IdempotencyConflict, "Idempotency-Key reused with a different request payload")
			}
			return err
		}
		if !rec.New {
			if rec.OperationID == nil {
				return apierr.New(apierr.IdempotencyConflict, "request with this Idempotency-Key is currently in progress")
			}
			_, etag, err := GetNodeTx(ctx, tx, nodeID)
			if err != nil {
				return err
			}
			op, err := GetOperationTx(ctx, tx, *rec.OperationID)
			if err != nil {
				return err
			}
			outOp = op
			outETag = etag
			return nil
		}

		beforeDetail, _, err := GetNodeTx(ctx, tx, nodeID)
		if err != nil {
			return err
		}
		if _, err := TransitionLifecycle(beforeDetail.LifecycleState, targetState); err != nil {
			return err
		}

		now := s.now()
		updates := map[string]any{
			"lifecycle_state":    string(targetState),
			"provisioning_state": string(api.ProvisioningStateFailed),
		}
		if retire {
			updates["retired_at"] = now
		}

		newVer, err := db.UpdateWithVersion(ctx, tx, "nodes", nodeID, expectedVer, updates)
		if err != nil {
			if errors.Is(err, db.ErrVersionConflict) {
				return versionConflictError(ctx, tx, "nodes", nodeID)
			}
			return err
		}

		// Revoke active incarnations, certificates, and unconsumed enrollment tokens.
		if _, err := tx.Exec(ctx, `
			UPDATE node_certificates
			SET revoked_at = $1
			WHERE incarnation_id IN (SELECT id FROM node_incarnations WHERE node_id = $2)
			  AND revoked_at IS NULL`, now, nodeID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE node_incarnations
			SET revoked_at = $1
			WHERE node_id = $2 AND revoked_at IS NULL`, now, nodeID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE node_enrollments
			SET state = 'revoked', revoked_at = $1
			WHERE node_id = $2 AND state = 'pending'`, now, nodeID); err != nil {
			return err
		}

		// Force exit publication_intent to disabled (§5.1).
		if _, err := tx.Exec(ctx, `
			UPDATE exits
			SET publication_intent = 'disabled', resource_version = resource_version + 1
			WHERE node_id = $1`, nodeID); err != nil {
			return err
		}

		nextGen := uint64(beforeDetail.ConfigGeneration.Desired) + 1
		op, err := CreateOperation(ctx, tx, CreateOperationParams{
			Kind:              opKind,
			Principal:         principal,
			ResourceType:      "node",
			ResourceID:        nodeID,
			ResourceVersion:   &newVer,
			RequestID:         requestID,
			DesiredGeneration: &nextGen,
			Now:               now,
		})
		if err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]any{
			"node_id":          nodeID.String(),
			"operation_id":     op.OperationId.String(),
			"lifecycle_state":  string(targetState),
			"reason":           reason,
			"resource_version": newVer,
			"occurred_at":      now.Format(time.RFC3339Nano),
		})
		if err := db.EnqueueOutbox(ctx, tx, db.OutboxEvent{
			AggregateID:      nodeID,
			AggregateType:    "node",
			AggregateVersion: newVer,
			EventType:        eventType,
			SchemaVersion:    1,
			Payload:          payload,
		}); err != nil {
			return err
		}

		beforeJSON, _ := json.Marshal(map[string]any{
			"lifecycle_state":    string(beforeDetail.LifecycleState),
			"publication_intent": string(beforeDetail.PublicationIntent),
			"resource_version":   beforeDetail.ResourceVersion.String(),
		})
		afterJSON, _ := json.Marshal(map[string]any{
			"lifecycle_state":    string(targetState),
			"publication_intent": string(api.PublicationIntentDisabled),
			"reason":             reason,
			"resource_version":   newVer,
		})
		if err := db.AppendAudit(ctx, tx, db.AuditEvent{
			Actor:         actorUUID.String(),
			ActorType:     actorType,
			Action:        opKind,
			Resource:      "node:" + nodeID.String(),
			Result:        "succeeded",
			RequestID:     requestID,
			OperationID:   &op.OperationId,
			SchemaVersion: 1,
			Before:        beforeJSON,
			After:         afterJSON,
		}); err != nil {
			return err
		}

		nodeIDCopy := nodeID
		if err := commandtx.CompleteIdempotent(ctx, tx, rec.ID, db.IdempotencyResult{
			OperationID:    &op.OperationId,
			ResourceType:   "node",
			ResourceID:     &nodeIDCopy,
			ResponseStatus: 202,
		}); err != nil {
			return err
		}

		outOp = op
		outETag = FormatETag(newVer)
		return nil
	})
	if err != nil {
		return api.Operation{}, "", err
	}
	return outOp, outETag, nil
}

func versionConflictError(ctx context.Context, tx pgx.Tx, table string, id uuid.UUID) error {
	var currentVer string
	err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT resource_version::text FROM %s WHERE id = $1`, table), id).Scan(&currentVer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.Newf(apierr.ResourceNotFound, "%s %s not found", table, id)
		}
		return err
	}
	return apierr.New(apierr.ResourceVersionConflict, "If-Match version does not match current resource_version").
		WithDetail("current_resource_version", currentVer)
}

func hashIdempotentRequest(target string, body any) ([]byte, error) {
	canon, err := canonjson.Marshal(map[string]any{
		"target": target,
		"body":   body,
	})
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canon)
	return sum[:], nil
}

// ParseU64 converts a uint64 to decstr.U64.
func ParseU64(v uint64) decstr.U64 {
	u, _ := decstr.Parse(strconv.FormatUint(v, 10))
	return u
}
