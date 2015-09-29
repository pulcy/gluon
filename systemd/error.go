package systemd

import (
	"github.com/juju/errgo"
)

var (
	ErrJobCanceled             = errgo.New("job has been canceled before it finished execution")
	ErrJobTimeout              = errgo.New("job timeout was reached")
	ErrJobFailed               = errgo.New("job failed")
	ErrJobSkipped              = errgo.New("job was skipped because it didn't apply to the units current state")
	ErrJobDependency           = errgo.New("job failed because of failed dependency")
	ErrJobExecutionTookTooLong = errgo.New("job execution took too long")
	ErrUnknownSystemdResponse  = errgo.New("received unknown systemd response")

	Mask = errgo.MaskFunc(errgo.Any)
)

func IsErrJobDependency(err error) bool {
	return errgo.Cause(err) == ErrJobDependency
}

