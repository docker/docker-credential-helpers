module github.com/docker/docker-credential-helpers

go 1.21

retract v0.9.0 // osxkeychain: a regression caused backward-incompatibility with earlier versions

require (
	github.com/danieljoos/wincred v1.2.2
	github.com/keybase/go-keychain v0.0.1
)

require golang.org/x/sys v0.20.0 // indirect
