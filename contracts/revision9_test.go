package contracts

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

func TestAPI01_Revision9FleetConfig(t *testing.T) {
	doc := loadDoc(t)
	// Revision 10 advances info.version to 1.6.0. This test still pins revision 9's decisions.
	if doc.Info.Version != "1.6.0" {
		t.Fatal("API version must be 1.6.0")
	}
	if n := len(operations(t)); n != 47 {
		t.Fatalf("got %d operations, want 47", n)
	}
	item := doc.Paths.Value("/admin/fleet-config")
	if item == nil || item.Get == nil || item.Put == nil {
		t.Fatal("missing /admin/fleet-config")
	}
	if item.Get.OperationID != "getFleetConfig" || item.Put.OperationID != "updateFleetConfig" {
		t.Fatalf("operation ids %s %s", item.Get.OperationID, item.Put.OperationID)
	}
	if item.Put.Responses.Status(200) == nil || item.Put.Responses.Status(202) != nil {
		t.Fatal("fleet-config update must be synchronous 200")
	}
	if item.Put.Responses.Status(412) == nil || item.Put.Responses.Status(428) == nil || item.Put.Responses.Status(422) == nil {
		t.Fatal("fleet-config update must expose 412, 422 and 428")
	}
	cfg := doc.Components.Schemas["FleetConfig"].Value
	if cfg.Properties["default_max_offline_lease_seconds"].Value.Enum[0] != float64(86400) {
		t.Fatal("default_max_offline_lease_seconds must be 86400")
	}
	seconds := cfg.Properties["max_offline_lease_seconds"].Value
	if seconds.Min == nil || *seconds.Min != 60 || seconds.Max == nil || *seconds.Max != 86400 {
		t.Fatal("max_offline_lease_seconds must be 60..86400")
	}
	desc := item.Get.Description + item.Put.Description + cfg.Description
	for _, phrase := range []string{"until renewal", "replay", "warn below 900", "fleet_config.update", "VALIDATION_FAILED"} {
		if !strings.Contains(desc, phrase) {
			t.Fatalf("description missing %q", phrase)
		}
	}
	raw := readFile(t, "examples/valid/FleetConfig.json")
	if err := validateComponent(t, "FleetConfig", raw); err != nil {
		t.Fatal(err)
	}
	var model api.FleetConfig
	if err := json.Unmarshal(raw, &model); err != nil {
		t.Fatal(err)
	}
	wantDefault := api.FleetConfigDefaultMaxOfflineLeaseSecondsN86400
	if model.MaxOfflineLeaseSeconds != 86400 || model.DefaultMaxOfflineLeaseSeconds == nil || *model.DefaultMaxOfflineLeaseSeconds != wantDefault || !wantDefault.Valid() || model.ResourceVersion != decstr.MustParse("1") {
		t.Fatalf("model %+v", model)
	}
	low := []byte(`{"max_offline_lease_seconds":59,"reason":"too short"}`)
	if err := validateComponent(t, "FleetConfigUpdate", low); err == nil {
		t.Fatal("59 seconds was accepted")
	}
}
