package main

import (
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/keyctl"
)

func main() {
	credentials.Serve(keyctl.Keyctl{})
}
