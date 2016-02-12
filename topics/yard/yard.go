package yard

import (
	"os"

	"github.com/juju/errgo"

	"github.com/pulcy/yard/templates"
	"github.com/pulcy/yard/topics"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	serviceTemplate = "templates/yard.service.tmpl"
	serviceName     = "yard.service"
	servicePath     = "/etc/systemd/system/" + serviceName
	yardPath        = "/home/core/bin/yard"

	fileMode = os.FileMode(0644)
)

func Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags, yardVersion string) error {
	if err := flags.SetupDefaults(yardVersion); err != nil {
		return maskAny(err)
	}
	if flags.YardPassphrase == "" {
		return errgo.New("yard-passphrase is missing")
	}
	if flags.PrivateRegistryUrl == "" {
		return errgo.New("private-registry-url is missing")
	}
	if flags.PrivateRegistryUserName == "" {
		return errgo.New("private-registry-username is missing")
	}
	if flags.PrivateRegistryPassword == "" {
		return errgo.New("private-registry-password is missing")
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
		if err := os.Remove(yardPath); err != nil {
			if !os.IsNotExist(err) {
				return maskAny(err)
			}
		}

		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}

		if err := deps.Systemd.Restart(serviceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createService(deps *topics.TopicDependencies, flags *topics.TopicFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	opts := struct {
		DiscoveryURL            string
		YardPassphrase          string
		YardImage               string
		PrivateClusterDevice    string
		PrivateRegistryUrl      string
		PrivateRegistryUserName string
		PrivateRegistryPassword string
	}{
		DiscoveryURL:            flags.DiscoveryURL,
		YardPassphrase:          flags.YardPassphrase,
		YardImage:               flags.YardImage,
		PrivateClusterDevice:    flags.PrivateClusterDevice,
		PrivateRegistryUrl:      flags.PrivateRegistryUrl,
		PrivateRegistryUserName: flags.PrivateRegistryUserName,
		PrivateRegistryPassword: flags.PrivateRegistryPassword,
	}
	changed, err := templates.Render(serviceTemplate, servicePath, opts, fileMode)
	return changed, maskAny(err)
}
