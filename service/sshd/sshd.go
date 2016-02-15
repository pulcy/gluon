package sshd

import (
	"os"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	confTemplate = "templates/sshd_config.tmpl"
	confName     = "sshd_config"
	confPath     = "/etc/ssh/" + confName

	fileMode = os.FileMode(0600)
)

func NewService() service.Service {
	return &sshdService{}
}

type sshdService struct{}

func (t *sshdService) Name() string {
	return "sshd"
}

func (t *sshdService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedConfig, err := createSshdConfig(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedConfig {
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createSshdConfig(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", confPath)
	os.Remove(confPath)
	changed, err := templates.Render(confTemplate, confPath, nil, fileMode)
	return changed, maskAny(err)
}
