package main

import (
	"github.com/calavera/docker-credential-helpers/credentials"
	"github.com/calavera/docker-credential-helpers/osxkeychain"
)

func main() {
	credentials.Serve(osxkeychain.New())
}
