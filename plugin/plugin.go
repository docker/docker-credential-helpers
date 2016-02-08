package plugin

import (
	"net/rpc"

	"github.com/calavera/docker-credential-helpers/credentials"
	"github.com/hashicorp/go-plugin"
)

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "DOCKER_CREDENTIAL_PLUGIN",
	MagicCookieValue: "nyzGgJQpfOYO$oUVHo4RsLaYaNmCqeWLEqZnZG}peMVq4nXdFp",
}

type credentialsPlugin struct {
	helper credentials.Helper
}

func (p *credentialsPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return p, nil
}

func (*credentialsPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return nil, nil
}

// Serve initializes the socket connection to a store helper.
func Serve(helper credentials.Helper) {
	pluginMap := map[string]plugin.Plugin{
		"credentials": &credentialsPlugin{helper},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
