package worker

import "testing"

func TestRecoveryModeT7_RegistrationCoverage(t *testing.T) {
	groups := map[string][]string{
		"allowed":  recoveryAllowedTypes,
		"filtered": {recoveryFilteredType},
		"waiting":  recoveryWaitingTypes,
	}
	membership := make(map[string]string)
	for group, types := range groups {
		for _, eventType := range types {
			if previous, exists := membership[eventType]; exists {
				t.Errorf("event %q occurs in %s and %s", eventType, previous, group)
			}
			membership[eventType] = group
		}
	}
	registration, err := productionRegistration()
	if err != nil {
		t.Fatal(err)
	}
	for eventType := range registration.Jobs {
		if _, exists := membership[eventType]; !exists {
			t.Errorf("registered event %q has no recovery classification", eventType)
		}
	}
}
