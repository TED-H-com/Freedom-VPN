# billing

Checkout provider truth projects a paid period

Status: pass

Run: 376e5f84-535e-4ddd-902b-985852d4691b

Source: 724ab19fd46a974a66d138ac0cddcd7f803bafb7

Spec: §9.2, §9.3, §19 AT03, §19 AT04

- checkout: 201
- hook_snapshot: 200
- hook_event: 200
- webhook_checkout.session.completed: 202
- webhook_checkout.session.completed: 202
- stored_rows_refund_0: {"anomalies": [], "paid_from": "2026-10-30T06:47:05+00:00", "paid_through": "2026-10-30T07:48:05+00:00", "refund_total": 0, "last_reconciled": true, "provider_status": "active", "entitlement_state": "active", "paid_invoice_count": 1, "cancel_at_period_end": false, "effective_service_end": "2026-10-30T07:48:05+00:00"}
- first_full_paid_period_reconciled: true
- refund_semantics: Paid invoice facts remain; a full refund revokes effective service rather than erasing history.
- period_start: 2026-10-30T06:47:05Z
- period_end: 2026-10-30T07:48:05Z
- hook_snapshot: 200
- hook_event: 200
- webhook_customer.subscription.updated: 202
- webhook_customer.subscription.updated: 202
- stored_rows_refund_250: {"anomalies": [], "paid_from": "2026-10-30T06:47:05+00:00", "paid_through": "2026-10-30T07:48:05+00:00", "refund_total": 250, "last_reconciled": true, "provider_status": "active", "entitlement_state": "cancel_at_period_end", "paid_invoice_count": 1, "cancel_at_period_end": true, "effective_service_end": "2026-10-30T07:48:05+00:00"}
- hook_snapshot: 200
- hook_event: 200
- webhook_charge.refunded: 202
- webhook_charge.refunded: 202
- stored_rows_refund_1000: {"anomalies": [], "paid_from": "2026-10-30T06:47:05+00:00", "paid_through": "2026-10-30T07:48:05+00:00", "refund_total": 1000, "last_reconciled": true, "provider_status": "canceled", "entitlement_state": "expired", "paid_invoice_count": 1, "cancel_at_period_end": true, "effective_service_end": null}
- invoice_rows_after_duplicate_deliveries: 1
- immutable_refund_rows: 2
