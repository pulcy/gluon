package weave

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
	installServiceName     = "install-weave.service"
	installServiceTemplate = "templates/install-weave.service.tmpl"
	installServicePath     = "/etc/systemd/system/" + installServiceName

	networkServiceName     = "10-weave.network"
	networkServiceTemplate = "templates/10-weave.network.tmpl"
	networkServicePath     = "/etc/systemd/network/10-weave.network"

	serviceName     = "weave.service"
	serviceTemplate = "templates/weave.service.tmpl"
	servicePath     = "/etc/systemd/system/" + serviceName

	weaveIPs = `$(IPS=""; while [ "$IPS" == "" ]; do IPS=$(/usr/bin/etcdctl member list | grep -v ${COREOS_PRIVATE_IPV4} | sed "s/\(.*\)clientURLs=http\:\/\/\([0-9\\.]*\).*/\2/g"); done; echo $IPS)`

	fileMode = os.FileMode(0664)
)

type weaveOptions struct {
	WeavePassword string
	WeaveIPS      string
}

type WeaveTopic struct {
}

func NewTopic() *WeaveTopic {
	return &WeaveTopic{}
}

func (t *WeaveTopic) Name() string {
	return "weave"
}

func (t *WeaveTopic) Setup(deps *topics.TopicDependencies) error {
	if err := createWeaveNetworkService(deps); err != nil {
		return maskAny(err)
	}
	if err := createWeaveInstallService(deps); err != nil {
		return maskAny(err)
	}
	if err := createWeaveService(deps); err != nil {
		return maskAny(err)
	}

	// reload unit files, that is, `systemctl daemon-reload`
	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	// start install-weave.service unit
	deps.Logger.Info("starting %s", installServiceName)
	if err := deps.Systemd.Start(installServiceName); err != nil {
		return maskAny(err)
	}

	// start install-weave.service unit
	deps.Logger.Info("starting %s", serviceName)
	if err := deps.Systemd.Start(serviceName); err != nil {
		return maskAny(err)
	}

	return nil
}

func createWeaveNetworkService(deps *topics.TopicDependencies) error {
	deps.Logger.Info("creating %s", networkServicePath)
	if err := templates.Render(networkServiceTemplate, networkServicePath, nil, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}

func createWeaveInstallService(deps *topics.TopicDependencies) error {
	deps.Logger.Info("creating %s", installServicePath)
	if err := templates.Render(installServiceTemplate, installServicePath, nil, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}

func createWeaveService(deps *topics.TopicDependencies) error {
	deps.Logger.Info("creating %s", servicePath)
	opts := weaveOptions{
		WeavePassword: "foo",
		WeaveIPS:      weaveIPs,
	}
	if err := templates.Render(serviceTemplate, servicePath, opts, 0600); err != nil {
		return maskAny(err)
	}

	return nil
}
