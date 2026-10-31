package e2e

// UsageSnapshot is the JSON CLI contract, decoded without importing node internals.
type UsageSnapshot struct {
	SchemaVersion       int            `json:"schema_version"`
	NodeID              string         `json:"node_id"`
	NodeIncarnationID   string         `json:"node_incarnation_id"`
	LedgerEpoch         *string        `json:"ledger_epoch"`
	MeterEpochs         []MeterEpoch   `json:"meter_epochs"`
	LastEventSeq        string         `json:"last_event_seq"`
	AckedThrough        string         `json:"acked_through"`
	UnackedCount        int            `json:"unacked_count"`
	UnackedPayloadBytes string         `json:"unacked_payload_bytes"`
	UnackedEvents       []UnackedEvent `json:"unacked_events"`
}

// MeterEpoch is one counter generation (§10.7). State is current or closed.
type MeterEpoch struct {
	MeterEpochID string `json:"meter_epoch_id"`
	State        string `json:"state"`
}

// UnackedEvent is one outbox row. PayloadSHA256 is the immutable hash of
// the stored canonical payload, the bytes the uploader resends (§10.5).
type UnackedEvent struct {
	EventSeq      string          `json:"event_seq"`
	PeriodIDs     []string        `json:"period_ids"`
	Records       []UnackedRecord `json:"records"`
	PayloadSHA256 string          `json:"payload_sha256"`
}

// UnackedRecord is one device row inside an unacknowledged event, copied
// from the stored payload.
type UnackedRecord struct {
	MemberID        string `json:"member_id"`
	DeviceID        string `json:"device_id"`
	BindingRevision string `json:"binding_revision"`
	PeriodID        string `json:"period_id"`
	UploadBytes     string `json:"upload_bytes"`
	DownloadBytes   string `json:"download_bytes"`
}
