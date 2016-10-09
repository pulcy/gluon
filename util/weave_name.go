package util

import (
	"fmt"
	"strings"
)

// WeaveNameFromMachineID converts a machine ID into a weave peer name.
func WeaveNameFromMachineID(machineID string) (string, error) {
	if len(machineID) < 12 {
		return "", fmt.Errorf("machineID '%s' is too short", machineID)
	}
	var parts []string
	for i := 0; i < 6; i++ {
		parts = append(parts, machineID[i*2:i*2+2])
	}
	return strings.Join(parts, ":"), nil
}
