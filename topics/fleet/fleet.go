package fleet

import (
	"fmt"
	"os"
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/topics"
	"github.com/pulcy/gluon/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	confTemplate = "templates/99-fleet.conf.tmpl"
	confName     = "99-fleet.conf"
	confPath     = "/etc/systemd/system/fleet.service.d/" + confName
	serviceName  = "fleet.service"

	fileMode = os.FileMode(0755)
)

type fleetTopic struct {
}

func NewTopic() *fleetTopic {
	return &fleetTopic{}
}

func (t *fleetTopic) Name() string {
	return "fleet"
}

func (t *fleetTopic) Defaults(flags *topics.TopicFlags) error {
	return nil
}

func (t *fleetTopic) Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	_, err := createFleetConf(deps, flags)
	if err != nil {
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

func createFleetConf(deps *topics.TopicDependencies, flags *topics.TopicFlags) (bool, error) {
	lines := []string{
		"[Service]",
		fmt.Sprintf("Environment=FLEET_METADATA=%s", flags.FleetMetadata),
		fmt.Sprintf("Environment=FLEET_PUBLIC_IP=%s", flags.PrivateIP),
	}

	changed, err := util.UpdateFile(confPath, []byte(strings.Join(lines, "\n")), fileMode)
	return changed, maskAny(err)
}
