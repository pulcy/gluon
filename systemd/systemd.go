package systemd

import (
	systemdPkg "github.com/coreos/go-systemd/dbus"
	"github.com/op/go-logging"
)

type SystemdClient struct {
	Logger *logging.Logger
}

func NewSystemdClient(logger *logging.Logger) *SystemdClient {
	return &SystemdClient{
		Logger: logger,
	}
}

func (sdc *SystemdClient) Reload() error {
	sdc.Logger.Debug("call systemd reload")

	conn, err := systemdPkg.New()
	if err != nil {
		return Mask(err)
	}

	if err := conn.Reload(); err != nil {
		sdc.Logger.Debug("systemd reload failed: %#v", err)
		return Mask(err)
	}

	return nil
}

// See http://godoc.org/github.com/coreos/go-systemd/dbus#Conn.StartUnit
func (sdc *SystemdClient) Start(unit string) error {
	sdc.Logger.Debug("call systemd start %s", unit)

	conn, err := systemdPkg.New()
	if err != nil {
		return Mask(err)
	}

	strChan := make(chan string, 1)
	if _, err := conn.StartUnit(unit, "replace", strChan); err != nil {
		sdc.Logger.Debug("systemd start failed: %#v", err)
		return Mask(err)
	}

	select {
	case res := <-strChan:
		sdc.Logger.Debug("systemd start responded %s", res)
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
			sdc.Logger.Error("unexpected systemd response: '%s'", res)
			return Mask(ErrUnknownSystemdResponse)
		}
	case <-timeoutJobExecution():
		return Mask(ErrJobExecutionTookTooLong)
	}

	return nil
}

// See http://godoc.org/github.com/coreos/go-systemd/dbus#Conn.RestartUnit
func (sdc *SystemdClient) Restart(unit string) error {
	sdc.Logger.Debug("call systemd restart %s", unit)

	conn, err := systemdPkg.New()
	if err != nil {
		return Mask(err)
	}

	strChan := make(chan string, 1)
	if _, err := conn.RestartUnit(unit, "replace", strChan); err != nil {
		sdc.Logger.Debug("systemd restart failed: %#v", err)
		return Mask(err)
	}

	select {
	case res := <-strChan:
		sdc.Logger.Debug("systemd restart responded %s", res)
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
			sdc.Logger.Error("unexpected systemd response: '%s'", res)
			return Mask(ErrUnknownSystemdResponse)
		}
	case <-timeoutJobExecution():
		return Mask(ErrJobExecutionTookTooLong)
	}

	return nil
}

// See http://godoc.org/github.com/coreos/go-systemd/dbus#Conn.StartUnit
func (sdc *SystemdClient) Stop(unit string) error {
	sdc.Logger.Debug("call systemd stop %s", unit)

	conn, err := systemdPkg.New()
	if err != nil {
		return Mask(err)
	}

	strChan := make(chan string, 1)
	if _, err := conn.StopUnit(unit, "replace", strChan); err != nil {
		sdc.Logger.Debug("systemd stop failed: %#v", err)
		return Mask(err)
	}

	select {
	case res := <-strChan:
		sdc.Logger.Debug("systemd stop responded %s", res)
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
			sdc.Logger.Error("unexpected systemd response: '%s'", res)
			return Mask(ErrUnknownSystemdResponse)
		}
	case <-timeoutJobExecution():
		return Mask(ErrJobExecutionTookTooLong)
	}

	return nil
}

// See http://godoc.org/github.com/coreos/go-systemd/dbus#Conn.EnableUnitFiles
func (sdc *SystemdClient) Enable(unit string) error {
	sdc.Logger.Debug("call systemd enable %s", unit)

	conn, err := systemdPkg.New()
	if err != nil {
		return Mask(err)
	}

	if _, _, err := conn.EnableUnitFiles([]string{unit}, false, false); err != nil {
		sdc.Logger.Debug("systemd enable failed: %#v", err)
		return Mask(err)
	}

	return nil
}

func (sdc *SystemdClient) Exists(unit string) (bool, error) {
	sdc.Logger.Debug("call systemd exists %s", unit)

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
