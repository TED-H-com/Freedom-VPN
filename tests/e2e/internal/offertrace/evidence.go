// Package offertrace defines content-free TEST ONLY delivery evidence.
package offertrace

import "time"

type Delivery struct {
	Generation    string    `json:"generation"`
	BodySHA256    string    `json:"body_sha256"`
	Bytes         int       `json:"bytes"`
	At            time.Time `json:"delivered_at"`
	WriteComplete bool      `json:"write_complete"`
}
type Local struct {
	Accepted       string    `json:"accepted_generation"`
	Applied        *string   `json:"applied_generation"`
	EnvelopeSHA256 string    `json:"envelope_sha256"`
	At             time.Time `json:"sampled_at"`
}
type ACK struct {
	ReportID   string    `json:"report_id"`
	Generation string    `json:"applied_generation"`
	Status     int       `json:"http_status"`
	At         time.Time `json:"received_at"`
}
type Evidence struct {
	CaptureID string    `json:"capture_id"`
	First     *Delivery `json:"first,omitempty"`
	Local     *Local    `json:"local,omitempty"`
	ACK       *ACK      `json:"ack,omitempty"`
	Error     string    `json:"error,omitempty"`
}
