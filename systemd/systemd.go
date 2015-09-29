package systemd

import (
	systemdPkg "github.com/coreos/go-systemd/dbus"
)

var vLogger = func(f string, v ...interface{}) {}

func Configure(vl func(f string, v ...interface{})) {
	vLogger = vl
}

type SystemdClient struct{}

func NewSystemdClient() (*SystemdClient, error) {
	return &SystemdClient{}, nil
}

func (sdc *SystemdClient) Reload() error {
	vLogger("  call SystemdClient.Reload()")

	conn, err := systemdPkg.New()
	if err != nil {
		return Mask(err)
	}

	if err := conn.Reload(); err != nil {
		return Mask(err)
	}

	return nil
}

// See http://godoc.org/github.com/coreos/go-systemd/dbus#Conn.StartUnit
func (sdc *SystemdClient) Start(unit string) error {
	vLogger("  call SystemdClient.Start(unit): %s", unit)

	conn, err := systemdPkg.New()
	if err != nil {
		return Mask(err)
	}

	strChan := make(chan string, 1)
	if _, err := conn.StartUnit(unit, "replace", strChan); err != nil {
		return Mask(err)
	}

	select {
	case res := <-strChan:
		switch res {
		case "done":
			return nil
		case "canceled":
			return Mask(ErrJobCanceled)
		case "timeout":
			return Mask(ErrJobTimeout)
		case "failed":
			// We need a start considered to be failed, when the unit is already running.
			return nil
		case "dependency":
			return Mask(ErrJobDependency)
		case "skipped":
			return Mask(ErrJobSkipped)
		default:
			// that should never happen
			vLogger("  unexpected systemd response: '%s'", res)
			return Mask(ErrUnknownSystemdResponse)
		}
	case <-timeoutJobExecution():
		return Mask(ErrJobExecutionTookTooLong)
	}

	return nil
}

// See http://godoc.org/github.com/coreos/go-systemd/dbus#Conn.StartUnit
func (sdc *SystemdClient) Stop(unit string) error {
	vLogger("  call SystemdClient.Stop(unit): %s", unit)

	conn, err := systemdPkg.New()
	if err != nil {
		return Mask(err)
	}

	strChan := make(chan string, 1)
	if _, err := conn.StopUnit(unit, "replace", strChan); err != nil {
		return Mask(err)
	}

	select {
	case res := <-strChan:
		switch res {
		case "done":
			return nil
		case "canceled":
			// In case the job that is stopped is canceled (because it was running),
			// it is stopped, so all good.
			return nil
		case "timeout":
			return Mask(ErrJobTimeout)
		case "failed":
			return Mask(ErrJobFailed)
		case "dependency":
			return Mask(ErrJobDependency)
		case "skipped":
			return Mask(ErrJobSkipped)
		default:
			// that should never happen
			vLogger("  unexpected systemd response: '%s'", res)
			return Mask(ErrUnknownSystemdResponse)
		}
	case <-timeoutJobExecution():
		return Mask(ErrJobExecutionTookTooLong)
	}

	return nil
}

func (sdc *SystemdClient) Exists(unit string) (bool, error) {
	vLogger("  call SystemdClient.Exists(unit): %s", unit)

	conn, err := systemdPkg.New()
	if err != nil {
		return false, Mask(err)
	}

	ustates, err := conn.ListUnits()
	if err != nil {
		return false, Mask(err)
	}

	for _, ustate := range ustates {
		if ustate.Name == unit {
			return true, nil
		}
	}

	return false, nil
}

