package docker

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/juju/errgo"

	"github.com/pulcy/yard/templates"
	"github.com/pulcy/yard/topics"
	"github.com/pulcy/yard/util"
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

// ConfigFile ~/.docker/config.json file info
// Taken from https://github.com/docker/docker/blob/master/cliconfig/config.go
type ConfigFile struct {
	AuthConfigs map[string]AuthConfig `json:"auths"`
}

// AuthConfig contains authorization information for connecting to a Registry
// Taken from https://github.com/docker/engine-api/blob/master/types/auth.go
type AuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth"`
	Email         string `json:"email"`
	ServerAddress string `json:"serveraddress,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}

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
	changedConfig, err := createDockerConfig(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedService, err := createDockerService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedConfig || changedService {
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(serviceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createDockerService(deps *topics.TopicDependencies, flags *topics.TopicFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	opts := struct {
		DockerIP string
	}{
		DockerIP: flags.DockerIP,
	}
	changed, err := templates.Render(serviceTemplate, servicePath, opts, fileMode)
	return changed, maskAny(err)
}

func createDockerConfig(deps *topics.TopicDependencies, flags *topics.TopicFlags) (bool, error) {
	if flags.PrivateRegistryPassword != "" && flags.PrivateRegistryUrl != "" && flags.PrivateRegistryUserName != "" {
		deps.Logger.Info("creating %s", rootConfigPath)
		// Load config file
		cf := ConfigFile{
			AuthConfigs: make(map[string]AuthConfig),
		}

		// Set authentication entries
		cf.AuthConfigs[flags.PrivateRegistryUrl] = AuthConfig{
			Auth: encodeAuth(AuthConfig{
				Username: flags.PrivateRegistryUserName,
				Password: flags.PrivateRegistryPassword,
				Email:    "",
			}),
		}

		// Save
		os.MkdirAll(filepath.Dir(rootConfigPath), 0700)
		raw, err := json.MarshalIndent(cf, "", "\t")
		if err != nil {
			return false, maskAny(err)
		}
		changed, err := util.UpdateFile(rootConfigPath, raw, 0600)
		return changed, maskAny(err)
	} else {
		deps.Logger.Warning("Skip creating .docker config")
	}

	return false, nil
}

// encodeAuth creates a base64 encoded string to containing authorization information
// Taken from https://github.com/docker/docker/blob/master/cliconfig/config.go
func encodeAuth(authConfig AuthConfig) string {
	authStr := authConfig.Username + ":" + authConfig.Password
	msg := []byte(authStr)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(encoded, msg)
	return string(encoded)
}
