package gluon

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
	serviceTemplate = "templates/gluon.service.tmpl"
	serviceName     = "gluon.service"
	servicePath     = "/etc/systemd/system/" + serviceName
	gluonPath       = "/home/core/bin/gluon"

	fileMode = os.FileMode(0644)
)

func NewService() service.Service {
	return &gluonService{}
}

type gluonService struct{}

func (t *gluonService) Name() string {
	return "gluon"
}

func (t *gluonService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	if err := flags.SetupDefaults(); err != nil {
		return maskAny(err)
	}
	if flags.DockerSubnet == "" {
		return errgo.New("docker-subnet is missing")
	}

	changedFlags, err := flags.Save()
	if err != nil {
		return maskAny(err)
	}

	changedService, err := createService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedFlags || changedService {
		if err := os.Remove(gluonPath); err != nil {
			if !os.IsNotExist(err) {
				return maskAny(err)
			}
		}

		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Enable(serviceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	opts := struct {
		GluonImage           string
		PrivateClusterDevice string
		DockerSubnet         string
	}{
		GluonImage:           flags.GluonImage,
		PrivateClusterDevice: flags.PrivateClusterDevice,
		DockerSubnet:         flags.DockerSubnet,
	}
	changed, err := templates.Render(serviceTemplate, servicePath, opts, fileMode)
	return changed, maskAny(err)
}
