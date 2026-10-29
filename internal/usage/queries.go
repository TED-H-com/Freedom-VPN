package usage

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// MemberUsageQuery specifies parameters for querying member usage (§10.9, §11.6, §14).
type MemberUsageQuery struct {
	MemberID    uuid.UUID
	Period      string // YYYY-MM (defaults to current UTC month if empty)
	Granularity Granularity
	FairUse     *api.FairUseStatus
}

// StreamWatermark captures the committed sequence watermark of a node incarnation stream
// at the time a member period total was read (§11.2 step 1).
type StreamWatermark struct {
	NodeID               uuid.UUID  `json:"node_id"`
	NodeIncarnationID    uuid.UUID  `json:"node_incarnation_id"`
	LedgerEpoch          uuid.UUID  `json:"ledger_epoch"`
	HighestContiguousSeq decstr.U64 `json:"highest_contiguous_seq"`
	MaxSeenSeq           decstr.U64 `json:"max_seen_seq"`
	PrefixProofHex       string     `json:"prefix_proof_hex"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// MemberPeriodTotal is the cross-node monthly usage total consumed by the fair-use
// evaluation pipeline (§11.2 step 1), including the ledger watermark used.
type MemberPeriodTotal struct {
	MemberID                 uuid.UUID         `json:"member_id"`
	Period                   string            `json:"period"`
	PeriodStart              time.Time         `json:"period_start"`
	PeriodEnd                time.Time         `json:"period_end"`
	MeteringVersion          string            `json:"metering_version"`
	UploadBytes              decstr.U64        `json:"upload_bytes"`
	DownloadBytes            decstr.U64        `json:"download_bytes"`
	TotalBytes               decstr.U64        `json:"total_bytes"`
	RawUploadBytes           decstr.U64        `json:"raw_upload_bytes"`
	RawDownloadBytes         decstr.U64        `json:"raw_download_bytes"`
	RawTotalBytes            decstr.U64        `json:"raw_total_bytes"`
	AdjustmentUploadDelta    string            `json:"adjustment_upload_delta"`
	AdjustmentDownloadDelta  string            `json:"adjustment_download_delta"`
	Estimated                bool              `json:"estimated"`
	Gap                      bool              `json:"gap"`
	ReconciliationInProgress bool              `json:"reconciliation_in_progress"`
	LastObservedAt           *time.Time        `json:"last_observed_at,omitempty"`
	AggregateAsOf            *time.Time        `json:"aggregate_as_of,omitempty"`
	LedgerEpoch              uuid.UUID         `json:"ledger_epoch"`
	ProjectionGeneration     decstr.U64        `json:"projection_generation"`
	Watermarks               []StreamWatermark `json:"watermarks"`
}

// DrillDownEntry is a single node incarnation + device attribution slice for a member and period.
// Usage is always attributed by historical device_bindings and node meter samples,
// never by login location or current exit (§10.2, §10.9).
type DrillDownEntry struct {
	NodeID            uuid.UUID   `json:"node_id"`
	NodeIncarnationID uuid.UUID   `json:"node_incarnation_id"`
	DeviceID          uuid.UUID   `json:"device_id"`
	Period            string      `json:"period"`
	Granularity       Granularity `json:"granularity"`
	PeriodStart       time.Time   `json:"period_start"`
	PeriodEnd         time.Time   `json:"period_end"`
	UploadBytes       decstr.U64  `json:"upload_bytes"`
	DownloadBytes     decstr.U64  `json:"download_bytes"`
	TotalBytes        decstr.U64  `json:"total_bytes"`
	Estimated         bool        `json:"estimated"`
	Gap               bool        `json:"gap"`
	LastObservedAt    time.Time   `json:"last_observed_at"`
}

// NodeUsageBreakdown aggregates a member's usage on a single node within a period.
type NodeUsageBreakdown struct {
	NodeID         uuid.UUID  `json:"node_id"`
	UploadBytes    decstr.U64 `json:"upload_bytes"`
	DownloadBytes  decstr.U64 `json:"download_bytes"`
	TotalBytes     decstr.U64 `json:"total_bytes"`
	Estimated      bool       `json:"estimated"`
	Gap            bool       `json:"gap"`
	LastObservedAt time.Time  `json:"last_observed_at"`
}

// DeviceUsageBreakdown aggregates a member's usage on a single device across all nodes within a period.
type DeviceUsageBreakdown struct {
	DeviceID       uuid.UUID  `json:"device_id"`
	UploadBytes    decstr.U64 `json:"upload_bytes"`
	DownloadBytes  decstr.U64 `json:"download_bytes"`
	TotalBytes     decstr.U64 `json:"total_bytes"`
	Estimated      bool       `json:"estimated"`
	Gap            bool       `json:"gap"`
	LastObservedAt time.Time  `json:"last_observed_at"`
}

// MemberDrillDown provides admin drill-down by node and device for a member and UTC month period.
type MemberDrillDown struct {
	MemberID     uuid.UUID              `json:"member_id"`
	Period       string                 `json:"period"`
	PeriodStart  time.Time              `json:"period_start"`
	PeriodEnd    time.Time              `json:"period_end"`
	Total        MemberPeriodTotal      `json:"total"`
	ByNode       []NodeUsageBreakdown   `json:"by_node"`
	ByDevice     []DeviceUsageBreakdown `json:"by_device"`
	ByNodeDevice []DrillDownEntry       `json:"by_node_device"`
	Adjustments  []Adjustment           `json:"adjustments"`
}

// GetMemberUsage returns the contract-shaped api.MemberUsage for a member and UTC month (§10.9, §11.6, §14).
func (s *Service) GetMemberUsage(ctx context.Context, q MemberUsageQuery) (api.MemberUsage, error) {
	if q.MemberID == uuid.Nil {
		return api.MemberUsage{}, apierr.Validation("member_id is required", apierr.FieldError{Field: "/member_id", Reason: "required"})
	}
	now := s.now()
	periodStr := q.Period
	if periodStr == "" {
		periodStr = MonthOf(now).ID
	}
	monthPeriod, err := ParseMonth(periodStr)
	if err != nil {
		return api.MemberUsage{}, apierr.Validation(err.Error(), apierr.FieldError{Field: "/period", Reason: "format"})
	}
	gran := q.Granularity
	if gran == "" {
		gran = GranularityMonth
	}
	if !gran.Valid() {
		return api.MemberUsage{}, apierr.Validation(fmt.Sprintf("unsupported granularity %q", gran), apierr.FieldError{Field: "/granularity", Reason: "enum"})
	}

	total, err := s.GetMemberPeriodTotal(ctx, q.MemberID, monthPeriod.ID)
	if err != nil {
		return api.MemberUsage{}, err
	}

	days := []api.UsageDay{}
	if gran == GranularityDay {
		days, err = s.queryUsageDays(ctx, q.MemberID, monthPeriod, uint64(total.ProjectionGeneration), total.ReconciliationInProgress)
		if err != nil {
			return api.MemberUsage{}, err
		}
	}

	var fairUse api.FairUseStatus
	if q.FairUse != nil {
		fairUse = *q.FairUse
		if fairUse.AsOf == nil {
			fairUse.AsOf = total.AggregateAsOf
		}
		if fairUse.ResetsAt == nil {
			endCopy := monthPeriod.End
			fairUse.ResetsAt = &endCopy
		}
	} else {
		tierZero := 0
		endCopy := monthPeriod.End
		fairUse = api.FairUseStatus{
			State:     api.FairUseStatusStateNormal,
			TierIndex: &tierZero,
			Applying:  false,
			AsOf:      total.AggregateAsOf,
			ResetsAt:  &endCopy,
		}
	}

	return api.MemberUsage{
		PeriodId:        monthPeriod.ID,
		PeriodStart:     monthPeriod.Start,
		PeriodEnd:       monthPeriod.End,
		Granularity:     api.MemberUsageGranularity(gran),
		MeteringVersion: api.MeteringVersion(total.MeteringVersion),
		Totals: api.UsageTotals{
			UploadBytes:   total.UploadBytes,
			DownloadBytes: total.DownloadBytes,
			TotalBytes:    total.TotalBytes,
		},
		Quality: api.UsageQualityFlags{
			Estimated:                total.Estimated,
			Gap:                      total.Gap,
			ReconciliationInProgress: total.ReconciliationInProgress,
		},
		LastObservedAt: total.LastObservedAt,
		AggregateAsOf:  total.AggregateAsOf,
		Days:           days,
		FairUse:        fairUse,
	}, nil
}

// GetMemberUsageSummary returns the compact api.MemberUsageSummary for member detail/list views.
func (s *Service) GetMemberUsageSummary(
	ctx context.Context,
	memberID uuid.UUID,
	periodStr string,
	fairUseTierIndex *int,
) (api.MemberUsageSummary, error) {
	total, err := s.GetMemberPeriodTotal(ctx, memberID, periodStr)
	if err != nil {
		return api.MemberUsageSummary{}, err
	}
	return api.MemberUsageSummary{
		PeriodId:         total.Period,
		TotalBytes:       total.TotalBytes,
		FairUseTierIndex: fairUseTierIndex,
		AggregateAsOf:    total.AggregateAsOf,
		Quality: api.UsageQualityFlags{
			Estimated:                total.Estimated,
			Gap:                      total.Gap,
			ReconciliationInProgress: total.ReconciliationInProgress,
		},
	}, nil
}

// GetMemberPeriodTotal returns a member's period total across all nodes for a UTC month,
// folding in append-only adjustments and capturing the ledger watermark used (§11.2 step 1).
func (s *Service) GetMemberPeriodTotal(ctx context.Context, memberID uuid.UUID, periodStr string) (MemberPeriodTotal, error) {
	if memberID == uuid.Nil {
		return MemberPeriodTotal{}, apierr.Validation("member_id is required", apierr.FieldError{Field: "/member_id", Reason: "required"})
	}
	now := s.now()
	if periodStr == "" {
		periodStr = MonthOf(now).ID
	}
	monthPeriod, err := ParseMonth(periodStr)
	if err != nil {
		return MemberPeriodTotal{}, apierr.Validation(err.Error(), apierr.FieldError{Field: "/period", Reason: "format"})
	}

	state, err := s.GetLedgerState(ctx)
	if err != nil {
		return MemberPeriodTotal{}, err
	}
	genStr := formatUint64(state.ActiveProjectionGeneration)

	rows, err := s.pool.Query(ctx, `
		SELECT upload_bytes::text,
		       download_bytes::text,
		       total_bytes::text,
		       metering_version,
		       estimated,
		       gap,
		       reconciliation_in_progress,
		       last_observed_at,
		       aggregate_as_of
		FROM usage_rollups
		WHERE projection_generation = $1::numeric
		  AND member_id = $2
		  AND period = $3
		  AND granularity = 'month'
	`, genStr, memberID, monthPeriod.ID)
	if err != nil {
		return MemberPeriodTotal{}, fmt.Errorf("usage: query member month rollup: %w", err)
	}
	defer rows.Close()

	var (
		rawUp       = new(big.Int)
		rawDown     = new(big.Int)
		rawTotal    = new(big.Int)
		meteringVer = MeteringVersionForwardAdmissionV1
		estimated   bool
		gap         bool
		reconciling bool
		lastObs     *time.Time
		aggAsOf     *time.Time
	)

	for rows.Next() {
		var (
			uText, dText, tText string
			mVer                string
			est, g, rec         bool
			lo, ao              time.Time
		)
		if err := rows.Scan(&uText, &dText, &tText, &mVer, &est, &g, &rec, &lo, &ao); err != nil {
			return MemberPeriodTotal{}, fmt.Errorf("usage: scan member month rollup: %w", err)
		}
		if u, ok := new(big.Int).SetString(uText, 10); ok {
			rawUp.Add(rawUp, u)
		}
		if d, ok := new(big.Int).SetString(dText, 10); ok {
			rawDown.Add(rawDown, d)
		}
		if t, ok := new(big.Int).SetString(tText, 10); ok {
			rawTotal.Add(rawTotal, t)
		}
		if mVer != "" {
			meteringVer = mVer
		}
		estimated = estimated || est
		gap = gap || g
		reconciling = reconciling || rec
		loUTC := lo.UTC()
		if lastObs == nil || loUTC.After(*lastObs) {
			lastObs = &loUTC
		}
		aoUTC := ao.UTC()
		if aggAsOf == nil || aoUTC.After(*aggAsOf) {
			aggAsOf = &aoUTC
		}
	}
	if err := rows.Err(); err != nil {
		return MemberPeriodTotal{}, fmt.Errorf("usage: iterate member month rollup: %w", err)
	}

	if state.Mode == LedgerModeReconciling {
		reconciling = true
	}

	// Check if any unrecoverable gap in usage_ledger_gaps affects this month for a node the member used.
	var hasLedgerGap bool
	err = s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM usage_ledger_gaps g
			WHERE $1 = ANY(g.affected_periods)
			  AND (
			      EXISTS (
			          SELECT 1 FROM usage_rollup_details d
			          WHERE d.projection_generation = $2::numeric
			            AND d.member_id = $3
			            AND d.node_id = g.node_id
			      )
			      OR NOT EXISTS (
			          SELECT 1 FROM usage_rollup_details d
			          WHERE d.projection_generation = $2::numeric
			            AND d.node_id = g.node_id
			      )
			  )
		)
	`, monthPeriod.ID, genStr, memberID).Scan(&hasLedgerGap)
	if err != nil {
		return MemberPeriodTotal{}, fmt.Errorf("usage: check ledger gaps for member: %w", err)
	}
	if hasLedgerGap {
		gap = true
	}

	// Fold in append-only adjustments for (member_id, monthPeriod.ID).
	var adjUpText, adjDownText string
	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(upload_bytes), 0)::text,
		       COALESCE(SUM(download_bytes), 0)::text
		FROM usage_adjustments
		WHERE member_id = $1 AND period = $2
	`, memberID, monthPeriod.ID).Scan(&adjUpText, &adjDownText)
	if err != nil {
		return MemberPeriodTotal{}, fmt.Errorf("usage: sum member adjustments: %w", err)
	}

	adjUp := new(big.Int)
	adjUp.SetString(adjUpText, 10)
	adjDown := new(big.Int)
	adjDown.SetString(adjDownText, 10)

	rawUpU64 := clampBigUint64(rawUp)
	rawDownU64 := clampBigUint64(rawDown)
	rawTotalU64 := clampBigUint64(rawTotal)

	effUpU64 := applyNumericDelta(rawUpU64, adjUp)
	effDownU64 := applyNumericDelta(rawDownU64, adjDown)
	var effTotalU64 uint64
	if effUpU64 > ^uint64(0)-effDownU64 {
		effTotalU64 = ^uint64(0)
	} else {
		effTotalU64 = effUpU64 + effDownU64
	}

	watermarks, err := s.queryWatermarksForMember(ctx, genStr, memberID, monthPeriod.ID)
	if err != nil {
		return MemberPeriodTotal{}, err
	}
	for _, wm := range watermarks {
		if aggAsOf == nil || wm.UpdatedAt.After(*aggAsOf) {
			t := wm.UpdatedAt.UTC()
			aggAsOf = &t
		}
	}

	return MemberPeriodTotal{
		MemberID:                 memberID,
		Period:                   monthPeriod.ID,
		PeriodStart:              monthPeriod.Start,
		PeriodEnd:                monthPeriod.End,
		MeteringVersion:          meteringVer,
		UploadBytes:              decstr.U64(effUpU64),
		DownloadBytes:            decstr.U64(effDownU64),
		TotalBytes:               decstr.U64(effTotalU64),
		RawUploadBytes:           decstr.U64(rawUpU64),
		RawDownloadBytes:         decstr.U64(rawDownU64),
		RawTotalBytes:            decstr.U64(rawTotalU64),
		AdjustmentUploadDelta:    adjUp.String(),
		AdjustmentDownloadDelta:  adjDown.String(),
		Estimated:                estimated,
		Gap:                      gap,
		ReconciliationInProgress: reconciling,
		LastObservedAt:           lastObs,
		AggregateAsOf:            aggAsOf,
		LedgerEpoch:              state.LedgerEpoch,
		ProjectionGeneration:     decstr.U64(state.ActiveProjectionGeneration),
		Watermarks:               watermarks,
	}, nil
}

// GetMemberDrillDown returns the admin breakdown of a member's usage by node and device
// for a UTC month period (§10.2, §10.9).
func (s *Service) GetMemberDrillDown(ctx context.Context, memberID uuid.UUID, periodStr string) (MemberDrillDown, error) {
	total, err := s.GetMemberPeriodTotal(ctx, memberID, periodStr)
	if err != nil {
		return MemberDrillDown{}, err
	}
	monthPeriod, err := ParseMonth(total.Period)
	if err != nil {
		return MemberDrillDown{}, apierr.Validation(err.Error(), apierr.FieldError{Field: "/period", Reason: "format"})
	}
	genStr := total.ProjectionGeneration.String()

	rows, err := s.pool.Query(ctx, `
		SELECT node_id, incarnation_id, device_id, period, granularity,
		       period_start, period_end,
		       upload_bytes::text, download_bytes::text, total_bytes::text,
		       estimated, gap, last_observed_at
		FROM usage_rollup_details
		WHERE projection_generation = $1::numeric
		  AND member_id = $2
		  AND period = $3
		  AND granularity = 'month'
		ORDER BY node_id ASC, incarnation_id ASC, device_id ASC
	`, genStr, memberID, monthPeriod.ID)
	if err != nil {
		return MemberDrillDown{}, fmt.Errorf("usage: query drill-down details: %w", err)
	}
	defer rows.Close()

	var byNodeDevice []DrillDownEntry
	nodeMap := make(map[uuid.UUID]*NodeUsageBreakdown)
	var nodeOrder []uuid.UUID
	deviceMap := make(map[uuid.UUID]*DeviceUsageBreakdown)
	var deviceOrder []uuid.UUID

	for rows.Next() {
		var (
			nodeID, incID, deviceID uuid.UUID
			per, granStr            string
			pStart, pEnd            time.Time
			uText, dText, tText     string
			est, g                  bool
			lo                      time.Time
		)
		if err := rows.Scan(
			&nodeID, &incID, &deviceID, &per, &granStr,
			&pStart, &pEnd,
			&uText, &dText, &tText,
			&est, &g, &lo,
		); err != nil {
			return MemberDrillDown{}, fmt.Errorf("usage: scan drill-down detail: %w", err)
		}
		u, err := parseNumericUint64(uText)
		if err != nil {
			return MemberDrillDown{}, err
		}
		d, err := parseNumericUint64(dText)
		if err != nil {
			return MemberDrillDown{}, err
		}
		t, err := parseNumericUint64(tText)
		if err != nil {
			return MemberDrillDown{}, err
		}

		byNodeDevice = append(byNodeDevice, DrillDownEntry{
			NodeID:            nodeID,
			NodeIncarnationID: incID,
			DeviceID:          deviceID,
			Period:            per,
			Granularity:       Granularity(granStr),
			PeriodStart:       pStart.UTC(),
			PeriodEnd:         pEnd.UTC(),
			UploadBytes:       decstr.U64(u),
			DownloadBytes:     decstr.U64(d),
			TotalBytes:        decstr.U64(t),
			Estimated:         est,
			Gap:               g,
			LastObservedAt:    lo.UTC(),
		})

		nb, ok := nodeMap[nodeID]
		if !ok {
			nb = &NodeUsageBreakdown{NodeID: nodeID}
			nodeMap[nodeID] = nb
			nodeOrder = append(nodeOrder, nodeID)
		}
		nb.UploadBytes += decstr.U64(u)
		nb.DownloadBytes += decstr.U64(d)
		nb.TotalBytes += decstr.U64(t)
		nb.Estimated = nb.Estimated || est
		nb.Gap = nb.Gap || g
		if lo.After(nb.LastObservedAt) {
			nb.LastObservedAt = lo.UTC()
		}

		dbEntry, ok := deviceMap[deviceID]
		if !ok {
			dbEntry = &DeviceUsageBreakdown{DeviceID: deviceID}
			deviceMap[deviceID] = dbEntry
			deviceOrder = append(deviceOrder, deviceID)
		}
		dbEntry.UploadBytes += decstr.U64(u)
		dbEntry.DownloadBytes += decstr.U64(d)
		dbEntry.TotalBytes += decstr.U64(t)
		dbEntry.Estimated = dbEntry.Estimated || est
		dbEntry.Gap = dbEntry.Gap || g
		if lo.After(dbEntry.LastObservedAt) {
			dbEntry.LastObservedAt = lo.UTC()
		}
	}
	if err := rows.Err(); err != nil {
		return MemberDrillDown{}, fmt.Errorf("usage: iterate drill-down details: %w", err)
	}

	byNode := make([]NodeUsageBreakdown, 0, len(nodeOrder))
	for _, id := range nodeOrder {
		byNode = append(byNode, *nodeMap[id])
	}
	byDevice := make([]DeviceUsageBreakdown, 0, len(deviceOrder))
	for _, id := range deviceOrder {
		byDevice = append(byDevice, *deviceMap[id])
	}

	adjs, err := s.ListAdjustments(ctx, memberID, monthPeriod.ID)
	if err != nil {
		return MemberDrillDown{}, err
	}

	return MemberDrillDown{
		MemberID:     memberID,
		Period:       monthPeriod.ID,
		PeriodStart:  monthPeriod.Start,
		PeriodEnd:    monthPeriod.End,
		Total:        total,
		ByNode:       byNode,
		ByDevice:     byDevice,
		ByNodeDevice: byNodeDevice,
		Adjustments:  adjs,
	}, nil
}

func (s *Service) queryUsageDays(
	ctx context.Context,
	memberID uuid.UUID,
	monthPeriod Period,
	generation uint64,
	reconciling bool,
) ([]api.UsageDay, error) {
	genStr := formatUint64(generation)
	rows, err := s.pool.Query(ctx, `
		SELECT period_start,
		       upload_bytes::text,
		       download_bytes::text,
		       total_bytes::text,
		       estimated,
		       gap,
		       reconciliation_in_progress
		FROM usage_rollups
		WHERE projection_generation = $1::numeric
		  AND member_id = $2
		  AND granularity = 'day'
		  AND period_start >= $3
		  AND period_end <= $4
		ORDER BY period_start ASC
	`, genStr, memberID, monthPeriod.Start, monthPeriod.End)
	if err != nil {
		return nil, fmt.Errorf("usage: query daily usage rows: %w", err)
	}
	defer rows.Close()

	days := make([]api.UsageDay, 0)
	for rows.Next() {
		var (
			pStart              time.Time
			uText, dText, tText string
			est, g, rec         bool
		)
		if err := rows.Scan(&pStart, &uText, &dText, &tText, &est, &g, &rec); err != nil {
			return nil, fmt.Errorf("usage: scan daily usage row: %w", err)
		}
		u, err := parseNumericUint64(uText)
		if err != nil {
			return nil, err
		}
		d, err := parseNumericUint64(dText)
		if err != nil {
			return nil, err
		}
		t, err := parseNumericUint64(tText)
		if err != nil {
			return nil, err
		}
		days = append(days, api.UsageDay{
			Date:          openapi_types.Date{Time: pStart.UTC()},
			UploadBytes:   decstr.U64(u),
			DownloadBytes: decstr.U64(d),
			TotalBytes:    decstr.U64(t),
			Quality: api.UsageQualityFlags{
				Estimated:                est,
				Gap:                      g,
				ReconciliationInProgress: rec || reconciling,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("usage: iterate daily usage rows: %w", err)
	}
	return days, nil
}

func (s *Service) queryWatermarksForMember(
	ctx context.Context,
	genStr string,
	memberID uuid.UUID,
	monthID string,
) ([]StreamWatermark, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.node_id, c.incarnation_id, c.ledger_epoch,
		       c.highest_contiguous_seq::text, c.max_seen_seq::text,
		       COALESCE(c.prefix_proof, ''::bytea), c.updated_at
		FROM usage_checkpoints c
		WHERE EXISTS (
		          SELECT 1 FROM usage_rollup_details d
		          WHERE d.projection_generation = $1::numeric
		            AND d.member_id = $2
		            AND d.period = $3
		            AND d.node_id = c.node_id
		      )
		   OR NOT EXISTS (
		          SELECT 1 FROM usage_rollup_details d
		          WHERE d.projection_generation = $1::numeric
		            AND d.member_id = $2
		            AND d.period = $3
		      )
		ORDER BY c.node_id ASC, c.incarnation_id ASC
	`, genStr, memberID, monthID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("usage: query stream watermarks: %w", err)
	}
	defer rows.Close()

	watermarks := make([]StreamWatermark, 0)
	for rows.Next() {
		var (
			wm              StreamWatermark
			highText, mText string
			proof           []byte
		)
		if err := rows.Scan(
			&wm.NodeID, &wm.NodeIncarnationID, &wm.LedgerEpoch,
			&highText, &mText, &proof, &wm.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("usage: scan stream watermark: %w", err)
		}
		high, err := parseNumericUint64(highText)
		if err != nil {
			return nil, err
		}
		mx, err := parseNumericUint64(mText)
		if err != nil {
			return nil, err
		}
		wm.HighestContiguousSeq = decstr.U64(high)
		wm.MaxSeenSeq = decstr.U64(mx)
		wm.PrefixProofHex = hex.EncodeToString(proof)
		wm.UpdatedAt = wm.UpdatedAt.UTC()
		watermarks = append(watermarks, wm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("usage: iterate stream watermarks: %w", err)
	}
	return watermarks, nil
}
