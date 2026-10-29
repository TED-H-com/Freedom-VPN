package agentstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	// StatusPending is an enrollment request persisted before the response.
	StatusPending = "pending"
	// StatusEnrolled is an enrollment whose certificate has been stored.
	StatusEnrolled = "enrolled"
	// OutboxPending is an observation the center has not accepted.
	OutboxPending = "pending"
	// OutboxAccepted means the center returned 202. That is not verification (§14.4).
	OutboxAccepted = "accepted"
	// OutboxConflict is a payload the center rejected as a conflict. It is kept.
	OutboxConflict = "conflict"
)

// Enrollment is the stable enrollment request (§6.1, §6.2).
type Enrollment struct {
	RequestID     uuid.UUID
	IncarnationID uuid.UUID
	CSRPEM        string
	Status        string
	NodeID        uuid.UUID
	CreatedAt     time.Time
	EnrolledAt    time.Time
}

// Identity is the public identity reference. The private key stays in the
// identity directory.
type Identity struct {
	NodeID        uuid.UUID
	IncarnationID uuid.UUID
	Serial        string
	NotAfter      time.Time
	PublicSHA256  string
}

// Desired is the last accepted desired-state envelope (§14.7).
type Desired struct {
	Generation  uint64
	Applied     uint64
	HasApplied  bool
	ETag        string
	Envelope    []byte
	IssuedAt    time.Time
	ExpiresAt   time.Time
	Publication string
}

// Lease is the lease the agent has accepted (§16.3). MonoSent is the
// monotonic time of the renewal request that first installed this lease id.
type Lease struct {
	ID              uuid.UUID
	IssuedAt        time.Time
	ExpiresAt       time.Time
	HighestIssuedAt time.Time
	Envelope        []byte
	MonoSent        time.Duration
	BootID          uuid.UUID
}

// Anchor is the persisted time anchor. A new boot must not treat it as live.
type Anchor struct {
	BootID  uuid.UUID
	Server  time.Time
	Mono    time.Duration
	Wall    time.Time
	Trusted bool
}

// Counter is one named-counter reading passed back to enforce on the next apply.
type Counter struct {
	Packets uint64
	Bytes   uint64
}

// OutboxItem is one observation waiting for, or already given, a 202.
type OutboxItem struct {
	ReportID uuid.UUID
	Payload  []byte
	Status   string
	Attempts int
}

// PutPendingEnrollment inserts the request ids. It fails if a row exists, so
// a retry cannot mint a second incarnation (§6.2, AT07).
func (s *Store) PutPendingEnrollment(ctx context.Context, requestID, incarnation uuid.UUID, at time.Time) error {
	if requestID == uuid.Nil || incarnation == uuid.Nil {
		return errors.New("agentstore: enrollment id is empty")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO enrollment
			(id, enrollment_request_id, node_incarnation_id, csr_pem, status, node_id, created_at, enrolled_at)
			VALUES (1, ?, ?, '', ?, '', ?, '')`,
			requestID.String(), incarnation.String(), StatusPending, fmtTime(at))
		return err
	})
}

// Enrollment returns the persisted request. ok is false when none exists.
func (s *Store) Enrollment(ctx context.Context) (Enrollment, bool, error) {
	var e Enrollment
	var req, inc, node, created, enrolled string
	err := s.db.QueryRowContext(ctx, `SELECT enrollment_request_id, node_incarnation_id, csr_pem, status, node_id, created_at, enrolled_at FROM enrollment WHERE id = 1`).
		Scan(&req, &inc, &e.CSRPEM, &e.Status, &node, &created, &enrolled)
	if errors.Is(err, sql.ErrNoRows) {
		return Enrollment{}, false, nil
	}
	if err != nil {
		return Enrollment{}, false, err
	}
	e.RequestID, err = uuid.Parse(req)
	if err != nil {
		return Enrollment{}, false, err
	}
	e.IncarnationID, err = uuid.Parse(inc)
	if err != nil {
		return Enrollment{}, false, err
	}
	if node != "" {
		e.NodeID, err = uuid.Parse(node)
		if err != nil {
			return Enrollment{}, false, err
		}
	}
	e.CreatedAt, err = parseTime(created)
	if err != nil {
		return Enrollment{}, false, err
	}
	e.EnrolledAt, err = parseTime(enrolled)
	if err != nil {
		return Enrollment{}, false, err
	}
	return e, true, nil
}

// SetEnrollmentCSR stores the CSR that will be sent. A later retry reuses it.
func (s *Store) SetEnrollmentCSR(ctx context.Context, csr string) error {
	if csr == "" {
		return errors.New("agentstore: empty csr")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE enrollment SET csr_pem = ? WHERE id = 1 AND status = ?`, csr, StatusPending)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("agentstore: enrollment csr was not stored")
		}
		return nil
	})
}

// CompleteEnrollment records the issued identity. It does not replace an
// enrollment that already finished.
func (s *Store) CompleteEnrollment(ctx context.Context, id Identity, at time.Time) error {
	if id.NodeID == uuid.Nil || id.IncarnationID == uuid.Nil || id.Serial == "" || id.PublicSHA256 == "" {
		return errors.New("agentstore: incomplete identity")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE enrollment SET status = ?, node_id = ?, enrolled_at = ?
			WHERE id = 1 AND status = ? AND node_incarnation_id = ?`,
			StatusEnrolled, id.NodeID.String(), fmtTime(at), StatusPending, id.IncarnationID.String())
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("agentstore: enrollment was not pending for this incarnation")
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO identity
			(id, node_id, node_incarnation_id, certificate_serial, certificate_not_after, public_key_sha256)
			VALUES (1, ?, ?, ?, ?, ?)`,
			id.NodeID.String(), id.IncarnationID.String(), id.Serial, fmtTime(id.NotAfter), id.PublicSHA256)
		return err
	})
}

// UpdateCertificate replaces the serial and expiry after a renewal (§6.4).
// The node id and incarnation stay the same.
func (s *Store) UpdateCertificate(ctx context.Context, serial string, notAfter time.Time) error {
	if serial == "" || notAfter.IsZero() {
		return errors.New("agentstore: incomplete certificate")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE identity SET certificate_serial = ?, certificate_not_after = ? WHERE id = 1`, serial, fmtTime(notAfter))
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("agentstore: certificate was not updated")
		}
		return nil
	})
}

// Identity returns the enrolled identity reference.
func (s *Store) Identity(ctx context.Context) (Identity, bool, error) {
	var id Identity
	var node, inc, notAfter string
	err := s.db.QueryRowContext(ctx, `SELECT node_id, node_incarnation_id, certificate_serial, certificate_not_after, public_key_sha256 FROM identity WHERE id = 1`).
		Scan(&node, &inc, &id.Serial, &notAfter, &id.PublicSHA256)
	if errors.Is(err, sql.ErrNoRows) {
		return Identity{}, false, nil
	}
	if err != nil {
		return Identity{}, false, err
	}
	id.NodeID, err = uuid.Parse(node)
	if err != nil {
		return Identity{}, false, err
	}
	id.IncarnationID, err = uuid.Parse(inc)
	if err != nil {
		return Identity{}, false, err
	}
	id.NotAfter, err = parseTime(notAfter)
	if err != nil {
		return Identity{}, false, err
	}
	return id, true, nil
}

// AcceptDesired stores a verified envelope. A generation older than the
// accepted one returns ErrStaleGeneration and writes nothing. An equal
// generation replaces the envelope (a re-signed document, §14.7) and does
// not clear the applied generation.
func (s *Store) AcceptDesired(ctx context.Context, d Desired) error {
	if d.Generation == 0 && len(d.Envelope) == 0 {
		return errors.New("agentstore: empty desired state")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		var current sql.NullInt64
		err := tx.QueryRowContext(ctx, `SELECT accepted_generation FROM desired WHERE id = 1`).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if current.Valid && uint64(current.Int64) > d.Generation {
			return ErrStaleGeneration
		}
		if !current.Valid {
			_, err = tx.ExecContext(ctx, `INSERT INTO desired
				(id, accepted_generation, applied_generation, etag, envelope, issued_at, expires_at, publication_intent)
				VALUES (1, ?, NULL, ?, ?, ?, ?, ?)`,
				d.Generation, d.ETag, d.Envelope, fmtTime(d.IssuedAt), fmtTime(d.ExpiresAt), d.Publication)
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE desired SET accepted_generation = ?, etag = ?, envelope = ?, issued_at = ?, expires_at = ?, publication_intent = ? WHERE id = 1`,
			d.Generation, d.ETag, d.Envelope, fmtTime(d.IssuedAt), fmtTime(d.ExpiresAt), d.Publication)
		return err
	})
}

// Desired returns the accepted desired state.
func (s *Store) Desired(ctx context.Context) (Desired, bool, error) {
	var d Desired
	var applied sql.NullInt64
	var issued, expires string
	err := s.db.QueryRowContext(ctx, `SELECT accepted_generation, applied_generation, etag, envelope, issued_at, expires_at, publication_intent FROM desired WHERE id = 1`).
		Scan(&d.Generation, &applied, &d.ETag, &d.Envelope, &issued, &expires, &d.Publication)
	if errors.Is(err, sql.ErrNoRows) {
		return Desired{}, false, nil
	}
	if err != nil {
		return Desired{}, false, err
	}
	if applied.Valid {
		d.HasApplied = true
		d.Applied = uint64(applied.Int64)
	}
	d.IssuedAt, err = parseTime(issued)
	if err != nil {
		return Desired{}, false, err
	}
	d.ExpiresAt, err = parseTime(expires)
	if err != nil {
		return Desired{}, false, err
	}
	return d, true, nil
}

// MarkApplied records the generation that enforce verified.
func (s *Store) MarkApplied(ctx context.Context, generation uint64) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE desired SET applied_generation = ? WHERE id = 1 AND accepted_generation >= ?`, generation, generation)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("agentstore: applied generation was not stored")
		}
		return nil
	})
}

// SaveLease replaces the lease row. Callers pass the original MonoSent when
// the lease id is unchanged (§16.3).
func (s *Store) SaveLease(ctx context.Context, l Lease) error {
	if l.ID == uuid.Nil || l.BootID == uuid.Nil || l.MonoSent <= 0 {
		return errors.New("agentstore: incomplete lease")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM lease WHERE id = 1`); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO lease
			(id, lease_id, issued_at, expires_at, highest_issued_at, envelope, mono_sent_ns, boot_id)
			VALUES (1, ?, ?, ?, ?, ?, ?, ?)`,
			l.ID.String(), fmtTime(l.IssuedAt), fmtTime(l.ExpiresAt), fmtTime(l.HighestIssuedAt),
			l.Envelope, int64(l.MonoSent), l.BootID.String())
		return err
	})
}

// Lease returns the stored lease.
func (s *Store) Lease(ctx context.Context) (Lease, bool, error) {
	var l Lease
	var id, issued, expires, highest, boot string
	var mono int64
	err := s.db.QueryRowContext(ctx, `SELECT lease_id, issued_at, expires_at, highest_issued_at, envelope, mono_sent_ns, boot_id FROM lease WHERE id = 1`).
		Scan(&id, &issued, &expires, &highest, &l.Envelope, &mono, &boot)
	if errors.Is(err, sql.ErrNoRows) {
		return Lease{}, false, nil
	}
	if err != nil {
		return Lease{}, false, err
	}
	l.ID, err = uuid.Parse(id)
	if err != nil {
		return Lease{}, false, err
	}
	l.BootID, err = uuid.Parse(boot)
	if err != nil {
		return Lease{}, false, err
	}
	l.IssuedAt, err = parseTime(issued)
	if err != nil {
		return Lease{}, false, err
	}
	l.ExpiresAt, err = parseTime(expires)
	if err != nil {
		return Lease{}, false, err
	}
	l.HighestIssuedAt, err = parseTime(highest)
	if err != nil {
		return Lease{}, false, err
	}
	l.MonoSent = time.Duration(mono)
	return l, true, nil
}

// SaveAnchor replaces the time anchor.
func (s *Store) SaveAnchor(ctx context.Context, a Anchor) error {
	if a.BootID == uuid.Nil || a.Server.IsZero() || a.Mono <= 0 {
		return errors.New("agentstore: incomplete anchor")
	}
	trusted := 0
	if a.Trusted {
		trusted = 1
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM anchor WHERE id = 1`); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO anchor (id, boot_id, server_unix_ns, mono_ns, wall_unix_ns, trusted) VALUES (1, ?, ?, ?, ?, ?)`,
			a.BootID.String(), a.Server.UnixNano(), int64(a.Mono), a.Wall.UnixNano(), trusted)
		return err
	})
}

// Anchor returns the stored anchor.
func (s *Store) Anchor(ctx context.Context) (Anchor, bool, error) {
	var a Anchor
	var boot string
	var server, mono, wall int64
	var trusted int
	err := s.db.QueryRowContext(ctx, `SELECT boot_id, server_unix_ns, mono_ns, wall_unix_ns, trusted FROM anchor WHERE id = 1`).
		Scan(&boot, &server, &mono, &wall, &trusted)
	if errors.Is(err, sql.ErrNoRows) {
		return Anchor{}, false, nil
	}
	if err != nil {
		return Anchor{}, false, err
	}
	a.BootID, err = uuid.Parse(boot)
	if err != nil {
		return Anchor{}, false, err
	}
	a.Server = time.Unix(0, server).UTC()
	a.Mono = time.Duration(mono)
	a.Wall = time.Unix(0, wall).UTC()
	a.Trusted = trusted == 1
	return a, true, nil
}

// QoS returns persisted class minors (§11.4).
func (s *Store) QoS(ctx context.Context) (map[uuid.UUID]uint16, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT member_id, minor FROM qos_assignment`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]uint16{}
	for rows.Next() {
		var id string
		var minor int
		if err := rows.Scan(&id, &minor); err != nil {
			return nil, err
		}
		u, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		if minor < 0 || minor > 0xffff {
			return nil, errors.New("agentstore: qos minor out of range")
		}
		out[u] = uint16(minor)
	}
	return out, rows.Err()
}

// SaveQoS replaces the class-minor map.
func (s *Store) SaveQoS(ctx context.Context, m map[uuid.UUID]uint16) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM qos_assignment`); err != nil {
			return err
		}
		for id, minor := range m {
			if _, err := tx.ExecContext(ctx, `INSERT INTO qos_assignment (member_id, minor) VALUES (?, ?)`, id.String(), minor); err != nil {
				return err
			}
		}
		return nil
	})
}

// Counters returns the last counter readings handed to enforce.
func (s *Store) Counters(ctx context.Context) (map[string]Counter, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, packets, bytes FROM counter_reading`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Counter{}
	for rows.Next() {
		var name string
		var c Counter
		if err := rows.Scan(&name, &c.Packets, &c.Bytes); err != nil {
			return nil, err
		}
		out[name] = c
	}
	return out, rows.Err()
}

// SaveCounters replaces counter readings.
func (s *Store) SaveCounters(ctx context.Context, m map[string]Counter) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM counter_reading`); err != nil {
			return err
		}
		for name, c := range m {
			if _, err := tx.ExecContext(ctx, `INSERT INTO counter_reading (name, packets, bytes) VALUES (?, ?, ?)`, name, c.Packets, c.Bytes); err != nil {
				return err
			}
		}
		return nil
	})
}

// Addresses returns overlay addresses installed by the last serving apply.
func (s *Store) Addresses(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT addr FROM applied_address ORDER BY addr`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// SaveAddresses replaces the applied address set.
func (s *Store) SaveAddresses(ctx context.Context, addrs []string) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM applied_address`); err != nil {
			return err
		}
		for _, a := range addrs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO applied_address (addr) VALUES (?)`, a); err != nil {
				return err
			}
		}
		return nil
	})
}

// Enqueue adds an observation. The same report id is a no-op so a crash
// between enqueue and send does not fork the id.
func (s *Store) Enqueue(ctx context.Context, id uuid.UUID, payload []byte, at time.Time) error {
	if id == uuid.Nil || len(payload) == 0 {
		return errors.New("agentstore: empty observation")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO outbox (report_id, payload, status, attempts, last_error, created_at, accepted_at)
			VALUES (?, ?, ?, 0, '', ?, '')
			ON CONFLICT(report_id) DO NOTHING`, id.String(), payload, OutboxPending, fmtTime(at))
		return err
	})
}

// Pending returns observations that still need a 202, oldest first.
func (s *Store) Pending(ctx context.Context) ([]OutboxItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT report_id, payload, status, attempts FROM outbox WHERE status = ? ORDER BY created_at, report_id`, OutboxPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutboxItem
	for rows.Next() {
		var item OutboxItem
		var id string
		if err := rows.Scan(&id, &item.Payload, &item.Status, &item.Attempts); err != nil {
			return nil, err
		}
		item.ReportID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// MarkAccepted records a 202. The row stays; acceptance is not verification.
func (s *Store) MarkAccepted(ctx context.Context, id uuid.UUID, at time.Time) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE outbox SET status = ?, accepted_at = ? WHERE report_id = ? AND status = ?`,
			OutboxAccepted, fmtTime(at), id.String(), OutboxPending)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("agentstore: observation was not pending")
		}
		return nil
	})
}

// MarkConflict keeps the payload and stops resending it.
func (s *Store) MarkConflict(ctx context.Context, id uuid.UUID) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE outbox SET status = ? WHERE report_id = ? AND status = ?`, OutboxConflict, id.String(), OutboxPending)
		return err
	})
}

// BumpAttempt records a failed send. The row stays pending.
func (s *Store) BumpAttempt(ctx context.Context, id uuid.UUID, reason string) error {
	if len(reason) > 200 {
		reason = reason[:200]
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE outbox SET attempts = attempts + 1, last_error = ? WHERE report_id = ?`, reason, id.String())
		return err
	})
}

// Meta reads one metadata value.
func (s *Store) Meta(ctx context.Context, key string) (string, bool, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// SetMeta upserts one metadata value inside a durable transaction.
func (s *Store) SetMeta(ctx context.Context, key, value string) error {
	if key == "" {
		return errors.New("agentstore: empty meta key")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO meta (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
		return err
	})
}
