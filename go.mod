module github.com/docker/docker-credential-helpers

go 1.24.1

retract (
	v0.9.1 // osxkeychain: a regression caused backward-incompatibility with earlier versions
	v0.9.0 // osxkeychain: a regression caused backward-incompatibility with earlier versions
)

require (
	github.com/danieljoos/wincred v1.2.3
	github.com/gopasspw/gopass v1.16.1
	github.com/keybase/go-keychain v0.0.1
)

require (
	al.essio.dev/pkg/shellescape v1.6.0 // indirect
	filippo.io/age v1.2.1 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/caspr-io/yamlpath v0.0.0-20200722075116-502e8d113a9b // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gopasspw/gitconfig v0.0.3 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/twpayne/go-pinentry/v4 v4.0.0 // indirect
	github.com/urfave/cli/v2 v2.27.7 // indirect
	github.com/xrash/smetrics v0.0.0-20250705151800-55b8f293f342 // indirect
	github.com/zalando/go-keyring v0.2.6 // indirect
	golang.org/x/crypto v0.44.0 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/term v0.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
