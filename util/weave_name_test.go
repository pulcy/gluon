package util

import "testing"

// Test WeaveNameFromMachineID converts a machine ID into a weave peer name.
func TestWeaveNameFromMachineID(t *testing.T) {
	name, err := WeaveNameFromMachineID("b7bdc73a4e4311ss80cabd7d1e4658a2")
	if err != nil {
		t.Fatalf("Expected success, got %#v", err)
	}
	expected := "b7:bd:c7:3a:4e:43"
	if name != expected {
		t.Errorf("Expected '%s', got '%s'", expected, name)
	}
}
