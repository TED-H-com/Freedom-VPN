package dnsfeeds

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// Options are process dependencies. No network I/O happens during construction.
type Options struct {
	Clock             func() time.Time
	Client            *http.Client
	UpdateCadence     time.Duration
	MaxRemovalPercent *int
	Metrics           *prometheus.Registry
}
type Service struct {
	pool    *pgxpool.Pool
	opts    Options
	removal int
	metrics *feedMetrics
}

func New(pool *pgxpool.Pool, opts Options) (*Service, error) {
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	if opts.UpdateCadence == 0 {
		opts.UpdateCadence = DefaultUpdateCadence
	}
	if opts.UpdateCadence < 5*time.Minute || opts.UpdateCadence > 7*24*time.Hour {
		return nil, errors.New("invalid feed cadence")
	}
	removal := DefaultMaxRemovalPercent
	if opts.MaxRemovalPercent != nil {
		removal = *opts.MaxRemovalPercent
	}
	if removal < 0 || removal > DefaultMaxRemovalPercent {
		return nil, errors.New("invalid feed removal bound")
	}
	s := &Service{pool: pool, opts: opts, removal: removal, metrics: newFeedMetrics(opts.Clock)}
	if opts.Metrics != nil {
		if err := opts.Metrics.Register(s.metrics); err != nil {
			return nil, err
		}
	}
	return s, nil
}

type attempt struct {
	revision uuid.UUID
	token    uuid.UUID
	payload  api.ThreatFeedPayload
	previous *Content
}

// Handle claims a short durable feed lease, fetches outside any transaction,
// then commits content, observations, audit and publication intents together.
// A rejected feed is a completed observation, not a successful refresh. Its next
// attempt is scheduled normally; storage errors remain retryable worker errors.
func (s *Service) Handle(ctx context.Context, event db.ClaimedEvent) (string, error) {
	if event.EventType != EventRefresh || event.SchemaVersion != 1 {
		return "", errors.New("invalid feed event")
	}
	var body struct {
		RevisionID uuid.UUID `json:"revision_id"`
	}
	if json.Unmarshal(event.Payload, &body) != nil || body.RevisionID == uuid.Nil || event.ID == uuid.Nil {
		return "", errors.New("invalid feed revision")
	}
	if s.pool == nil {
		return "", errors.New("feed storage required")
	}
	now := s.opts.Clock().UTC()
	a, err := s.begin(ctx, body.RevisionID, event.ID, now)
	if err != nil {
		return "", err
	}
	if a == nil {
		return "dns-feed:unchanged:" + body.RevisionID.String(), nil
	}
	content, reason := Fetch(ctx, s.opts.Client, a.payload, a.previous, now, s.removal)
	// A canceled fetch must not leave an unreported half attempt. Lease expiry
	// permits recovery; never claim a successful observation on canceled context.
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	changed, err := s.finish(ctx, a, event, content, reason, s.opts.Clock().UTC())
	if err != nil {
		return "", err
	}
	if !changed {
		return "dns-feed:superseded:" + body.RevisionID.String(), nil
	}
	{
		result := reason
		if result == "" {
			result = "success"
		}
		s.metrics.total.WithLabelValues(a.payload.FeedCode, result).Inc()
		if content != nil {
			s.metrics.set(a.revision, a.payload.FeedCode, s.opts.Clock().UTC())
		}
	}
	if reason != "" {
		return "dns-feed:failed:" + reason + ":" + body.RevisionID.String(), nil
	}
	return "dns-feed:refreshed:" + body.RevisionID.String(), nil
}

func (s *Service) begin(ctx context.Context, rev, event uuid.UUID, now time.Time) (*attempt, error) {
	var a *attempt
	err := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		a = nil
		var raw []byte
		err := tx.QueryRow(ctx, `SELECT r.payload FROM policy_revisions r JOIN policy_definitions d ON d.active_revision_id=r.id WHERE r.id=$1 AND r.type='threat_feed' FOR SHARE OF d`, rev).Scan(&raw)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		candidate := &attempt{revision: rev, token: uuid.New()}
		if err := json.Unmarshal(raw, &candidate.payload); err != nil {
			return errors.New("invalid stored feed")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO dns_feed_refresh_state(feed_revision_id) VALUES($1) ON CONFLICT DO NOTHING`, rev); err != nil {
			return err
		}
		var hash *string
		var next, until *time.Time
		var lastEvent *uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT last_good_content,next_attempt_at,attempt_until,last_event_id FROM dns_feed_refresh_state WHERE feed_revision_id=$1 FOR UPDATE`, rev).Scan(&hash, &next, &until, &lastEvent); err != nil {
			return err
		}
		if lastEvent != nil && *lastEvent == event {
			return nil
		}
		if until != nil && now.Before(*until) {
			return errors.New("feed refresh in progress")
		}
		if next != nil && now.Before(*next) {
			return nil
		}
		if hash != nil {
			var entries []byte
			var previous Content
			if err := tx.QueryRow(ctx, `SELECT content_sha256,entries FROM dns_feed_contents WHERE feed_revision_id=$1 AND content_sha256=$2`, rev, *hash).Scan(&previous.SHA256, &entries); err != nil {
				return err
			}
			if err := json.Unmarshal(entries, &previous.Entries); err != nil {
				return err
			}
			candidate.previous = &previous
		}
		_, err = tx.Exec(ctx, `UPDATE dns_feed_refresh_state SET last_attempt_at=$2,attempt_token=$3,attempt_until=$4 WHERE feed_revision_id=$1`, rev, now, candidate.token, now.Add(time.Minute))
		if err == nil {
			a = candidate
		}
		return err
	})
	return a, err
}

func (s *Service) finish(ctx context.Context, a *attempt, event db.ClaimedEvent, c *Content, reason string, now time.Time) (bool, error) {
	committed := false
	err := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		committed = false
		// Recheck activation under a lock: an in-flight fetch cannot resurrect a
		// revision replaced while the HTTP request was running.
		var active bool
		err := tx.QueryRow(ctx, `SELECT coalesce(active_revision_id=$1,false) FROM policy_definitions WHERE id=(SELECT definition_id FROM policy_revisions WHERE id=$1) FOR SHARE`, a.revision).Scan(&active)
		if err != nil {
			return err
		}
		if !active {
			return nil
		}
		var token *uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT attempt_token FROM dns_feed_refresh_state WHERE feed_revision_id=$1 FOR UPDATE`, a.revision).Scan(&token); err != nil {
			return err
		}
		if token == nil || *token != a.token {
			return errors.New("feed attempt superseded")
		}
		// When invoked by the dispatcher, fence the original outbox claim as well.
		if event.Worker != "" {
			var owns bool
			if err := tx.QueryRow(ctx, `SELECT lease_owner=$2 AND lease_token=$3 AND lease_until>clock_timestamp() AND delivered_at IS NULL FROM outbox_events WHERE id=$1 FOR UPDATE`, event.ID, event.Worker, event.LeaseToken).Scan(&owns); err != nil {
				return err
			}
			if !owns {
				return db.ErrLeaseLost
			}
		}
		cadence := max(s.opts.UpdateCadence, time.Duration(a.payload.UpdateIntervalSeconds)*time.Second)
		action, result := "dns.feed.refresh_failed", api.AuditEventResultFailed
		after := map[string]any{"reason_code": reason}
		fields := []string{"reason_code"}
		if c == nil {
			_, err = tx.Exec(ctx, `UPDATE dns_feed_refresh_state SET last_error_code=$2,consecutive_failures=consecutive_failures+1,next_attempt_at=$3,attempt_token=NULL,attempt_until=NULL,last_event_id=$4 WHERE feed_revision_id=$1`, a.revision, reason, now.Add(cadence), event.ID)
		} else {
			entries, e := json.Marshal(c.Entries)
			if e != nil {
				return e
			}
			metrics, e := json.Marshal(c.Metrics)
			if e != nil {
				return e
			}
			contentID, e := uuid.NewV7()
			if e != nil {
				return e
			}
			_, err = tx.Exec(ctx, `INSERT INTO dns_feed_contents(feed_revision_id,content_sha256,size_bytes,rule_count,entries,parse_metrics,fetched_at,kind,id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`, a.revision, c.SHA256, c.Size, len(c.Entries), entries, metrics, now, c.Kind, contentID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE dns_feed_refresh_state SET last_good_content=$2,last_success_at=$3,last_error_code=NULL,consecutive_failures=0,next_attempt_at=$4,attempt_token=NULL,attempt_until=NULL,last_event_id=$5 WHERE feed_revision_id=$1`, a.revision, c.SHA256, now, now.Add(cadence), event.ID)
			action, result = "dns.feed.refreshed", api.AuditEventResultSucceeded
			after = map[string]any{"content_sha256": c.SHA256, "rule_count": len(c.Entries)}
			fields = []string{"content_sha256", "rule_count"}
			if err == nil && (a.previous == nil || a.previous.SHA256 != c.SHA256) {
				// Durable fan-out includes profiles still waiting for their first content.
				_, err = tx.Exec(ctx, `INSERT INTO outbox_events(aggregate_id,aggregate_type,aggregate_version,event_type,schema_version,payload)
     SELECT d.id,'policy_definition',d.generation,'dns.publish',1,
      jsonb_build_object('type','dns_profile','definition_id',d.id,'revision_id',r.id,'generation',d.generation::text,'action','feed_refresh')
     FROM policy_definitions d JOIN policy_revisions r ON r.id=d.active_revision_id
     WHERE r.type='dns_profile' AND r.payload->'threat_feed_revision_ids' @> jsonb_build_array($1::text)`, a.revision.String())
			}
		}
		if err != nil {
			return err
		}
		ev, err := audit.NewEvent(audit.Entry{Actor: audit.SystemActor, Action: action, Resource: api.ResourceRef{Type: api.ResourceTypePolicyRevision, Id: a.revision}, Result: result, After: after, Fields: fields})
		if err != nil {
			return err
		}
		if err := audit.AppendEvent(ctx, tx, ev); err != nil {
			return err
		}
		committed = true
		return nil
	})
	return committed, err
}

// Schedule repairs the durable cadence chain. A transaction advisory lock
// coalesces concurrent worker processes; only active revisions are enqueued.
func (s *Service) Schedule(ctx context.Context) error {
	if s.pool == nil {
		return errors.New("feed storage required")
	}
	now := s.opts.Clock().UTC()
	err := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(544054)`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO outbox_events(aggregate_id,aggregate_type,aggregate_version,event_type,schema_version,payload)
   SELECT r.id,'policy_revision',d.generation,'dns.feed.refresh',1,jsonb_build_object('revision_id',r.id)
   FROM policy_definitions d JOIN policy_revisions r ON r.id=d.active_revision_id
   LEFT JOIN dns_feed_refresh_state s ON s.feed_revision_id=r.id
   WHERE r.type='threat_feed' AND (s.next_attempt_at IS NULL OR s.next_attempt_at <= $1)
    AND (s.attempt_until IS NULL OR s.attempt_until <= $1)
    AND NOT EXISTS(SELECT 1 FROM outbox_events e WHERE e.event_type='dns.feed.refresh' AND e.aggregate_id=r.id AND e.delivered_at IS NULL AND e.dead_lettered_at IS NULL)`, now)
		return err
	})
	if err != nil {
		return err
	}
	return s.observeAges(ctx)
}

func (s *Service) observeAges(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT r.id,r.payload->>'feed_code',s.last_success_at FROM policy_definitions d JOIN policy_revisions r ON r.id=d.active_revision_id LEFT JOIN dns_feed_refresh_state s ON s.feed_revision_id=r.id WHERE r.type='threat_feed'`)
	if err != nil {
		return err
	}
	defer rows.Close()
	ages := map[uuid.UUID]feedAge{}
	for rows.Next() {
		var id uuid.UUID
		var code string
		var at *time.Time
		if err := rows.Scan(&id, &code, &at); err != nil {
			return err
		}
		value := feedAge{code: code}
		if at != nil {
			value.at = *at
		}
		ages[id] = value
	}
	if err := rows.Err(); err != nil {
		return err
	}
	s.metrics.mu.Lock()
	s.metrics.ages = ages
	s.metrics.mu.Unlock()
	return nil
}
