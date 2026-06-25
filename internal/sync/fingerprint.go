package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// MachineFingerprint generates a unique fingerprint for the current machine.
// It combines hostname, OS, architecture, and a machine-specific identifier.
func MachineFingerprint() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	machineID, err := getMachineID()
	if err != nil {
		// Fall back to hostname-based fingerprint if machine ID is unavailable.
		machineID = hostname
	}

	parts := []string{
		hostname,
		runtime.GOOS,
		runtime.GOARCH,
		machineID,
	}

	h := sha256.Sum256([]byte(strings.Join(parts, ":")))
	return hex.EncodeToString(h[:]), nil
}

// getMachineID reads the machine-id from the system.
func getMachineID() (string, error) {
	// Linux: /etc/machine-id or /var/lib/dbus/machine-id
	// macOS: use IOPlatformUUID via system_profiler (handled differently)
	paths := []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}

	return "", fmt.Errorf("machine-id not found")
}
