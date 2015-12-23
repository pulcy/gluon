package docker

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/cliconfig"
	"github.com/juju/errgo"

	"arvika.pulcy.com/pulcy/yard/templates"
	"arvika.pulcy.com/pulcy/yard/topics"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	serviceTemplate = "templates/docker.service.tmpl"
	serviceName     = "docker.service"
	servicePath     = "/etc/systemd/system/" + serviceName
	rootConfigPath  = "/root/.docker/config"

	fileMode = os.FileMode(0755)
)

type DockerTopic struct {
}

func NewTopic() *DockerTopic {
	return &DockerTopic{}
}

func (t *DockerTopic) Name() string {
	return "docker"
}

func (t *DockerTopic) Defaults(flags *topics.TopicFlags) error {
	return nil
}

func (t *DockerTopic) Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	if err := createDockerConfig(deps, flags); err != nil {
		return maskAny(err)
	}

	if err := createDockerService(deps, flags); err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Restart(serviceName); err != nil {
		return maskAny(err)
	}

	return nil
}

func createDockerService(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	deps.Logger.Info("creating %s", servicePath)
	opts := struct {
		DockerIP string
	}{
		DockerIP: flags.DockerIP,
	}
	if err := templates.Render(serviceTemplate, servicePath, opts, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}

func createDockerConfig(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	if flags.PrivateRegistryPassword != "" && flags.PrivateRegistryUrl != "" && flags.PrivateRegistryUserName != "" {
		deps.Logger.Info("creating %s", rootConfigPath)
		// Load config file
		cf, err := cliconfig.Load(filepath.Dir(rootConfigPath))
		if err != nil {
			return maskAny(err)
		}

		// Set authentication entries
		cf.AuthConfigs[flags.PrivateRegistryUrl] = types.AuthConfig{
			Username: flags.PrivateRegistryUserName,
			Password: flags.PrivateRegistryPassword,
			Email:    "",
		}

		// Save
		if err := cf.Save(); err != nil {
			return maskAny(err)
		}
	} else {
		deps.Logger.Warning("Skip creating .docker config")
	}

	return nil
}
