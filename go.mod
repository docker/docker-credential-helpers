module github.com/docker/docker-credential-helpers

go 1.21

require (
	github.com/danieljoos/wincred v1.2.2
	github.com/keybase/dbus v0.0.0-20220506165403-5aa21ea2c23a
	github.com/keybase/go-keychain v0.0.1
)

require (
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
)

replace github.com/keybase/dbus => github.com/godbus/dbus/v5 v5.1.0
