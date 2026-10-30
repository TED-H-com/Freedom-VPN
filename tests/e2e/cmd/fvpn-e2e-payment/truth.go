package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
)

// snapshot is TEST ONLY mutable provider truth. Its financial identity is fixed
// by the first snapshot; subsequent changes model cancellation and refunds.
type refund struct {
	ID     string
	Amount int64
}

type snapshot struct {
	CheckoutID string `json:"checkout_id"`
	Start      int64  `json:"period_start"`
	End        int64  `json:"period_end"`
	Amount     int64  `json:"amount_minor"`
	Currency   string `json:"currency"`
	Status     string `json:"status"`
	Paid       bool   `json:"paid"`
	Cancel     bool   `json:"cancel_at_period_end"`
	Refund     int64  `json:"refund_minor"`
}

func (s *server) hook(w http.ResponseWriter, r *http.Request) {
	if s.hookKey == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-E2E-Secret")), []byte(s.hookKey)) != 1 {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	if r.Method != "POST" {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	decode := func(v any) bool {
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
		d.DisallowUnknownFields()
		return d.Decode(v) == nil && d.Decode(new(any)) == io.EOF
	}
	switch r.URL.Path {
	case "/e2e/snapshot":
		var v snapshot
		if !decode(&v) {
			writeJSON(w, 400, map[string]string{"error": "payload"})
			return
		}
		c, ok := s.sessions[v.CheckoutID]
		if !ok || strings.HasPrefix(v.CheckoutID, "idem:") {
			writeJSON(w, 404, map[string]string{"error": "checkout missing"})
			return
		}
		_, currencyErr := billing.CurrencyDigits(v.Currency)
		if v.Start <= 0 || v.End <= v.Start || v.Amount <= 0 || v.Amount > 1_000_000_000 || v.Refund < 0 || v.Refund > v.Amount || (v.Status != "active" && v.Status != "canceled") || currencyErr != nil || (!v.Paid && v.Refund != 0) {
			writeJSON(w, 400, map[string]string{"error": "snapshot values"})
			return
		}
		if old, ok := s.truth[c.ID]; ok && (old.Start != v.Start || old.End != v.End || old.Amount != v.Amount || old.Currency != v.Currency || old.Refund > v.Refund || (old.Paid && !v.Paid)) {
			writeJSON(w, 409, map[string]string{"error": "immutable invoice or refund regression"})
			return
		}
		old := s.truth[c.ID]
		if v.Refund > old.Refund {
			id, err := newID("re_e2e_")
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": "id"})
				return
			}
			if s.refunds == nil {
				s.refunds = map[string][]refund{}
			}
			s.refunds[c.ID] = append(s.refunds[c.ID], refund{ID: id, Amount: v.Refund - old.Refund})
		}
		s.truth[c.ID] = v
		writeJSON(w, 200, map[string]any{"checkout_id": c.ID, "subscription_id": "sub_" + c.ID, "invoice_id": "in_" + c.ID, "customer_id": c.Customer})
	case "/e2e/event":
		var v struct {
			CheckoutID string `json:"checkout_id"`
			Type       string `json:"type"`
		}
		if !decode(&v) {
			writeJSON(w, 400, map[string]string{"error": "payload"})
			return
		}
		c, ok := s.sessions[v.CheckoutID]
		_, hasTruth := s.truth[v.CheckoutID]
		if !ok || !hasTruth {
			writeJSON(w, 404, map[string]string{"error": "snapshot missing"})
			return
		}
		obj := map[string]any{"id": c.ID, "customer": c.Customer, "subscription": "sub_" + c.ID, "client_reference_id": c.Checkout, "metadata": map[string]string{"fvpn_checkout_id": c.Checkout}}
		switch v.Type {
		case "checkout.session.completed":
		case "customer.subscription.updated", "customer.subscription.deleted":
			obj["id"] = "sub_" + c.ID
		case "invoice.paid":
			obj["id"] = "in_" + c.ID
			obj["parent"] = map[string]any{"subscription_details": map[string]string{"subscription": "sub_" + c.ID}}
		case "charge.refunded":
			obj["id"] = "ch_" + c.ID
		default:
			writeJSON(w, 400, map[string]string{"error": "event type"})
			return
		}
		id, err := newID("evt_e2e_")
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "id"})
			return
		}
		raw, _ := json.Marshal(map[string]any{"id": id, "type": v.Type, "api_version": billing.StripeAPIVersion, "livemode": false, "data": map[string]any{"object": obj}})
		timestamp := fmt.Sprint(time.Now().Unix())
		mac := hmac.New(sha256.New, []byte(s.webhookKey))
		mac.Write([]byte(timestamp + "."))
		mac.Write(raw)
		// Body is a string: decoding and re-encoding an object would change its HMAC.
		writeJSON(w, 200, map[string]string{"body": string(raw), "signature": "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))})
	default:
		writeJSON(w, 404, map[string]string{"error": "missing"})
	}
}

func (s *server) readTruth(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, q := r.URL.Path, r.URL.Query()
	data := []any{}
	known := path == "/v1/subscriptions" || path == "/v1/invoices" || path == "/v1/invoice_payments" || path == "/v1/refunds"
	for id, v := range s.truth {
		c := s.sessions[id]
		sub, invoice, charge := "sub_"+id, "in_"+id, "ch_"+id
		subscription := map[string]any{"id": sub, "customer": c.Customer, "status": v.Status, "livemode": false, "cancel_at_period_end": v.Cancel, "metadata": map[string]string{"fvpn_checkout_id": c.Checkout}, "items": map[string]any{"has_more": false, "data": []any{map[string]any{"price": map[string]string{"id": c.Price}, "quantity": 1}}}}
		if path == "/v1/subscriptions/"+sub {
			writeJSON(w, 200, subscription)
			return
		}
		if path == "/v1/subscriptions" && (q.Get("customer") == "" || q.Get("customer") == c.Customer) {
			data = append(data, subscription)
		}
		if path == "/v1/invoices" && v.Paid && (q.Get("subscription") == "" || q.Get("subscription") == sub) {
			data = append(data, map[string]any{"id": invoice, "customer": c.Customer, "status": "paid", "currency": strings.ToLower(v.Currency), "amount_paid": v.Amount, "parent": map[string]any{"subscription_details": map[string]string{"subscription": sub}}})
		}
		if path == "/v1/invoices/"+invoice+"/lines" {
			known = true
			data = append(data, map[string]any{"id": "il_" + id, "amount": v.Amount, "quantity": 1, "period": map[string]int64{"start": v.Start, "end": v.End}, "pricing": map[string]any{"price_details": map[string]string{"price": c.Price}}, "parent": map[string]any{"type": "subscription_item_details", "subscription_item_details": map[string]any{"subscription": sub, "proration": false}}})
		}
		if path == "/v1/invoice_payments" && v.Paid && (q.Get("invoice") == "" || q.Get("invoice") == invoice) && (q.Get("payment[charge]") == "" || q.Get("payment[charge]") == charge) {
			data = append(data, map[string]any{"id": "ip_" + id, "invoice": invoice, "status": "paid", "payment": map[string]string{"type": "charge", "charge": charge}})
		}
		if path == "/v1/refunds" && v.Refund > 0 && q.Get("charge") == charge {
			for _, refund := range s.refunds[id] {
				data = append(data, map[string]any{"id": refund.ID, "amount": refund.Amount, "currency": strings.ToLower(v.Currency), "status": "succeeded"})
			}
		}
	}
	if !known {
		writeJSON(w, 404, map[string]string{"error": "missing"})
		return
	}
	writeJSON(w, 200, map[string]any{"object": "list", "data": data, "has_more": false})
}
