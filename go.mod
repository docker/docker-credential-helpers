module github.com/docker/docker-credential-helpers

go 1.21

retract v0.9.0 // osxkeychain: a regression caused backward-incompatibility with earlier versions

require (
	github.com/danieljoos/wincred v1.2.2
	github.com/keybase/go-keychain v0.0.1
	gotest.tools/v3 v3.5.2
)

require (
	github.com/google/go-cmp v0.5.9 // indirect
	golang.org/x/sys v0.20.0 // indirect
)
