package main

import (
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/gopass"
)

func main() {
	credentials.Serve(gopass.Gopass{})
}
