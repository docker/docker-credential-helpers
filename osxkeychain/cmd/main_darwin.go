package main

import (
	"github.com/calavera/docker-credential-helpers/osxkeychain"
	"github.com/calavera/docker-credential-helpers/plugin"
)

func main() {
	plugin.Serve(osxkeychain.New())
}
