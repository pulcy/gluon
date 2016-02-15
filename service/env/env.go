package env

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
	bashrcTemplate = "templates/bashrc.tmpl"
	bashrcPath     = "/home/core/.bashrc"

	fileMode = os.FileMode(0644)
)

func NewService() service.Service {
	return &envService{}
}

type envService struct{}

func (t *envService) Name() string {
	return "env"
}

func (t *envService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	if err := createBashrc(deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

func createBashrc(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	deps.Logger.Info("creating %s", bashrcPath)
	os.Remove(bashrcPath)
	if _, err := templates.Render(bashrcTemplate, bashrcPath, nil, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}
