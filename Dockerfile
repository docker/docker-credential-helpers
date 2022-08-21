# syntax=docker/dockerfile:1

ARG GO_VERSION=1.18.5
ARG XX_VERSION=1.1.2
ARG OSXCROSS_VERSION=11.3-r7-alpine
ARG GOLANGCI_LINT_VERSION=v1.47.3

ARG PKG=github.com/docker/docker-credential-helpers

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:${XX_VERSION} AS xx

# osxcross contains the MacOSX cross toolchain for xx
FROM crazymax/osxcross:${OSXCROSS_VERSION} AS osxcross

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS gobase
COPY --from=xx / /
RUN apk add --no-cache clang file git lld llvm pkgconf rsync
ENV GOFLAGS="-mod=vendor"
ENV CGO_ENABLED="1"
WORKDIR /src

FROM gobase AS vendored
RUN --mount=target=/context \
    --mount=target=.,type=tmpfs  \
    --mount=target=/go/pkg/mod,type=cache <<EOT
  set -e
  rsync -a /context/. .
  go mod tidy
  go mod vendor
  mkdir /out
  cp -r go.mod go.sum vendor /out
EOT

FROM scratch AS vendor-update
COPY --from=vendored /out /

FROM vendored AS vendor-validate
RUN --mount=type=bind,target=.,rw <<EOT
  set -e
  rsync -a /context/. .
  git add -A
  rm -rf vendor
  cp -rf /out/* .
  if [ -n "$(git status --porcelain -- go.mod go.sum vendor)" ]; then
    echo >&2 'ERROR: Vendor result differs. Please vendor your package with "make vendor"'
    git status --porcelain -- go.mod go.sum vendor
    exit 1
  fi
EOT

FROM golangci/golangci-lint:${GOLANGCI_LINT_VERSION}-alpine AS golangci-lint
FROM gobase AS lint
RUN apk add musl-dev gcc libsecret-dev pass
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=from=golangci-lint,source=/usr/bin/golangci-lint,target=/usr/bin/golangci-lint \
    golangci-lint run ./...

FROM gobase AS version
ARG PKG
RUN --mount=target=. \
    VERSION=$(git describe --match 'v[0-9]*' --dirty='.m' --always --tags); \
    echo "-s -w -X ${PKG}/credentials.Version=${VERSION}" | tee /tmp/.ldflags; \
    echo -n "${VERSION}" | tee /tmp/.version;

FROM gobase AS base
ARG TARGETPLATFORM
RUN xx-apk add musl-dev gcc libsecret-dev pass

FROM base AS test
RUN xx-apk add gnome-keyring gpg-agent gnupg-gpgconf pass
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
  set -e
  cp -r .github/workflows/fixtures /root/.gnupg
  gpg-connect-agent "RELOADAGENT" /bye
  gpg --import --batch --yes /root/.gnupg/7D851EB72D73BDA0.key
  echo -e "trust\n5\ny" | gpg --batch --no-tty --command-fd 0 --edit-key 7D851EB72D73BDA0
  gpg-connect-agent "PRESET_PASSPHRASE 3E2D1142AA59E08E16B7E2C64BA6DDC773B1A627 -1 77697468207374757069642070617373706872617365" /bye
  gpg-connect-agent "KEYINFO 3E2D1142AA59E08E16B7E2C64BA6DDC773B1A627" /bye
  gpg-connect-agent "PRESET_PASSPHRASE BA83FC8947213477F28ADC019F6564A956456163 -1 77697468207374757069642070617373706872617365" /bye
  gpg-connect-agent "KEYINFO BA83FC8947213477F28ADC019F6564A956456163" /bye
  pass init 7D851EB72D73BDA0

  mkdir /out
  xx-go test -short -v -coverprofile=/out/coverage.txt -covermode=atomic ./...
  xx-go tool cover -func=/out/coverage.txt
EOT

FROM scratch AS test-coverage
COPY --from=test /out /

FROM base AS build-linux
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,from=version,source=/tmp/.ldflags,target=/tmp/.ldflags <<EOT
  set -ex
  mkdir /out
  xx-go build -ldflags "$(cat /tmp/.ldflags)" -o /out/docker-credential-pass-${TARGETOS}-${TARGETARCH}${TARGETVARIANT} ./pass/cmd/
  xx-verify /out/docker-credential-pass-${TARGETOS}-${TARGETARCH}${TARGETVARIANT}
  xx-go build -ldflags "$(cat /tmp/.ldflags)" -o /out/docker-credential-secretservice-${TARGETOS}-${TARGETARCH}${TARGETVARIANT} ./secretservice/cmd/
  xx-verify /out/docker-credential-secretservice-${TARGETOS}-${TARGETARCH}${TARGETVARIANT}
EOT

FROM base AS build-darwin
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,from=osxcross,src=/osxsdk,target=/xx-sdk \
    --mount=type=bind,from=version,source=/tmp/.ldflags,target=/tmp/.ldflags <<EOT
  set -ex
  mkdir /out
  xx-go install std
  xx-go build -ldflags "$(cat /tmp/.ldflags)" -o /out/docker-credential-osxkeychain-${TARGETARCH}${TARGETVARIANT} ./osxkeychain/cmd/
  xx-verify /out/docker-credential-osxkeychain-${TARGETARCH}${TARGETVARIANT}

  xx-go build -ldflags "$(cat /tmp/.ldflags)" -o /out/docker-credential-pass-${TARGETOS}-${TARGETARCH}${TARGETVARIANT} ./pass/cmd/
  xx-verify /out/docker-credential-pass-${TARGETOS}-${TARGETARCH}${TARGETVARIANT}
EOT

FROM base AS build-windows
ARG TARGETARCH
ARG TARGETVARIANT
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,from=version,source=/tmp/.ldflags,target=/tmp/.ldflags <<EOT
  set -ex
  mkdir /out
  xx-go build -ldflags "$(cat /tmp/.ldflags)" -o /out/docker-credential-wincred-${TARGETARCH}${TARGETVARIANT}.exe ./wincred/cmd/
  xx-verify /out/docker-credential-wincred-${TARGETARCH}${TARGETVARIANT}.exe
EOT

FROM build-$TARGETOS AS build

FROM scratch AS binaries
COPY --from=build /out /
