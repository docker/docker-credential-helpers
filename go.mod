module github.com/docker/docker-credential-helpers

go 1.23.6

retract (
	v0.9.1 // osxkeychain: a regression caused backward-incompatibility with earlier versions
	v0.9.0 // osxkeychain: a regression caused backward-incompatibility with earlier versions
)

require (
	github.com/danieljoos/wincred v1.2.2
	github.com/gopasspw/gopass v1.15.15
	github.com/keybase/go-keychain v0.0.1
)

require (
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	filippo.io/age v1.2.1-0.20240618131852-7eedd929a6cf // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/ProtonMail/go-crypto v1.1.2 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/caspr-io/yamlpath v0.0.0-20200722075116-502e8d113a9b // indirect
	github.com/cloudflare/circl v1.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/go-github/v61 v61.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/nbutton23/zxcvbn-go v0.0.0-20210217022336-fa2cb2858354 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/twpayne/go-pinentry v0.3.0 // indirect
	github.com/urfave/cli/v2 v2.27.5 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/zalando/go-keyring v0.2.6 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/exp v0.0.0-20241108190413-2d47ceb2692f // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/term v0.28.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
