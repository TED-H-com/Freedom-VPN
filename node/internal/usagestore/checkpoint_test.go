package usagestore

import (
	"encoding/json"
	"testing"
)

func TestD125_EmptyBarrierPreservesLegacyFourMembers(t *testing.T) {
	raw, err := json.Marshal(MeterCheckpoint{ClosedMeterEpochID: "019a0000-0000-7000-8000-000000000001", LastDurableEventSeq: 17, CentralAckedThrough: 12})
	want := `{"closed_meter_epoch_id":"019a0000-0000-7000-8000-000000000001","last_durable_event_seq":"17","central_acked_through":"12","new_meter_epoch_id":""}`
	if err != nil || string(raw) != want {
		t.Fatalf("legacy bytes changed: %s %v", raw, err)
	}
}
