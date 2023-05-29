module github.com/docker/docker-credential-helpers

go 1.19

require (
	github.com/danieljoos/wincred v1.2.0
	github.com/keybase/dbus v0.0.0-20220506165403-5aa21ea2c23a
	github.com/keybase/go-keychain v0.0.0-20230523030712-b5615109f100
)

require (
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
)

replace github.com/keybase/dbus => github.com/godbus/dbus/v5 v5.1.0
