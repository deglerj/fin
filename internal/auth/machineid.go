package auth

import (
	"os"
	"strings"
)

type MachineIDProvider interface {
	MachineID() (string, error)
}

type DefaultMachineID struct{}

func (DefaultMachineID) MachineID() (string, error) {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		b, err := os.ReadFile(p)
		if err == nil {
			return strings.TrimSpace(string(b)), nil
		}
	}
	return os.Hostname()
}
