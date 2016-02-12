package etcd2

import (
	"fmt"
	"os"
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/yard/templates"
	"github.com/pulcy/yard/topics"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	confTemplate = "templates/99-etcd2.conf.tmpl"
	confName     = "99-etcd2.conf"
	confPath     = "/run/systemd/system/etcd2.service.d/" + confName
	serviceName  = "etcd2.service"

	fileMode = os.FileMode(0755)
)

func ConfigureEtcd2(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	changedConf, err := createEtcd2Conf(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedConf {
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(serviceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createEtcd2Conf(deps *topics.TopicDependencies, flags *topics.TopicFlags) (bool, error) {
	deps.Logger.Info("creating %s", confPath)
	members, err := flags.GetClusterMembers()
	if err != nil {
		deps.Logger.Warning("GetClusterMembers failed: %v", err)
	}
	opts := struct {
		DiscoveryURL        string
		InitialCluster      string
		InitialClusterState string
	}{}
	if len(members) == 0 {
		// Use discovery url
		opts.DiscoveryURL = flags.DiscoveryURL
	} else {
		items := []string{}
		for _, cm := range members {
			items = append(items, fmt.Sprintf("%s=http://%s:2380", cm.MachineID, cm.PrivateIP))
		}
		opts.InitialCluster = strings.Join(items, ",")
		opts.InitialClusterState = "existing"
	}
	changed, err := templates.Render(confTemplate, confPath, opts, fileMode)
	return changed, maskAny(err)
}
