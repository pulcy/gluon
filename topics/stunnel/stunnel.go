package stunnel

import (
	"os"

	"github.com/juju/errgo"

	"arvika.pulcy.com/iggi/yard/templates"
	"arvika.pulcy.com/iggi/yard/topics"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	pemImage     = "ewoutp/igsp:0.0.3"
	stunnelImage = "ewoutp/stunnel:latest"

	registryServiceName     = "stunnel-registry.service"
	registryServiceTemplate = "templates/stunnel-registry.service.tmpl"
	registryServicePath     = "/etc/systemd/system/" + registryServiceName

	registryConfTemplate = "templates/stunnel-registry.conf.tmpl"
	registryConfPath     = "/etc/stunnel/stunnel-registry.conf"

	fileMode = os.FileMode(0664)
)

type StunnelTopic struct {
}

func NewTopic() *StunnelTopic {
	return &StunnelTopic{}
}

func (t *StunnelTopic) Name() string {
	return "stunnel"
}

func (t *StunnelTopic) Defaults(flags *topics.TopicFlags) error {
	if flags.StunnelPemPassphrase == "" {
		return errgo.New("StunnelPemPassphrase is not set")
	}
	return nil
}

func (t *StunnelTopic) Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	if err := createStunnelRegistryConf(deps, flags); err != nil {
		return maskAny(err)
	}
	if err := createStunnelRegistryService(deps, flags); err != nil {
		return maskAny(err)
	}

	// reload unit files, that is, `systemctl daemon-reload`
	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	// start install-weave.service unit
	deps.Logger.Info("starting %s", registryServiceName)
	if err := deps.Systemd.Start(registryServiceName); err != nil {
		return maskAny(err)
	}

	return nil
}

func createStunnelRegistryConf(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	opts := struct {
		PrivateRegistryHost string
		PrivateRegistryPort int
	}{
		PrivateRegistryHost: "arvika.pulcy.com",
		PrivateRegistryPort: 36702,
	}
	deps.Logger.Info("creating %s", registryConfPath)
	if err := templates.Render(registryConfTemplate, registryConfPath, opts, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}

func createStunnelRegistryService(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	opts := struct {
		PemImage      string
		Image         string
		PemPassphrase string
		ConfPath      string
	}{
		PemImage:      pemImage,
		Image:         stunnelImage,
		PemPassphrase: flags.StunnelPemPassphrase,
		ConfPath:      registryConfPath,
	}
	deps.Logger.Info("creating %s", registryServicePath)
	if err := templates.Render(registryServiceTemplate, registryServicePath, opts, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}
