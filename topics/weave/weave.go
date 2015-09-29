package weave

import (
	"io/ioutil"
	"os"
	"strconv"
	"text/template"

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
	if err := createWeaveInstallService(); err != nil {
		return maskAny(err)
	}
	if err := createWeaveService(); err != nil {
		return maskAny(err)
	}

	// reload unit files, that is, `systemctl daemon-reload`
	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	// start install-weave.service unit
	if err := deps.Systemd.Start(installServiceName); err != nil {
		return maskAny(err)
	}

	// start install-weave.service unit
	if err := deps.Systemd.Start(serviceName); err != nil {
		return maskAny(err)
	}

	return nil
}

func createWeaveInstallService() error {
	installService, err := templates.Asset(installServiceTemplate)
	if err != nil {
		return maskAny(err)
	}

	if err := ioutil.WriteFile(installServicePath, installService, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}

func createWeaveService() error {
	weaveService, err := templates.Asset(serviceTemplate)
	if err != nil {
		return maskAny(err)
	}

	// parse template
	var tmpl *template.Template
	tmpl, err = template.New(serviceName).Parse(string(weaveService))
	if err != nil {
		return maskAny(err)
	}
	f, err := os.Create(servicePath)
	if err != nil {
		return maskAny(err)
	}
	// write unit file to host
	opts := weaveOptions{
		WeavePassword: "foo",
		WeaveIPS:      escape(weaveIPs),
	}
	err = tmpl.Execute(f, opts)
	if err != nil {
		return maskAny(err)
	}
	f.Chmod(0600)

	return nil
}

func escape(s string) string {
	s = strconv.Quote(s)
	return s[1 : len(s)-1]
}
