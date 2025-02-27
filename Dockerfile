# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23.6
ARG DEBIAN_VERSION=bookworm

ARG XX_VERSION=1.6.1
ARG OSXCROSS_VERSION=11.3-r7-debian
ARG GOLANGCI_LINT_VERSION=v1.64.5
ARG DEBIAN_FRONTEND=noninteractive

ARG PACKAGE=github.com/docker/docker-credential-helpers

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:${XX_VERSION} AS xx

# osxcross contains the MacOSX cross toolchain for xx
FROM crazymax/osxcross:${OSXCROSS_VERSION} AS osxcross

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-${DEBIAN_VERSION} AS gobase
COPY --from=xx / /
ARG DEBIAN_FRONTEND
RUN apt-get update && apt-get install -y --no-install-recommends clang dpkg-dev file git lld llvm make pkg-config rsync
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
RUN --mount=target=/context \
    --mount=target=.,type=tmpfs <<EOT
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

FROM golangci/golangci-lint:${GOLANGCI_LINT_VERSION} AS golangci-lint
FROM gobase AS lint
ARG DEBIAN_FRONTEND
RUN apt-get install -y binutils gcc libc6-dev libgcc-11-dev libsecret-1-dev pkg-config
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=from=golangci-lint,source=/usr/bin/golangci-lint,target=/usr/bin/golangci-lint \
    golangci-lint run ./...

FROM gobase AS base
ARG TARGETPLATFORM
ARG DEBIAN_FRONTEND
RUN xx-apt-get install -y binutils gcc libc6-dev libgcc-11-dev libsecret-1-dev pkg-config

FROM base AS test
ARG DEBIAN_FRONTEND
RUN xx-apt-get install -y dbus-x11 gnome-keyring gpg-agent gpgconf libsecret-1-dev pass
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
  set -e
  cp -r .github/workflows/fixtures /root/.gnupg
  gpg-connect-agent "RELOADAGENT" /bye
  gpg --import --batch --yes /root/.gnupg/7D851EB72D73BDA0.key
  gpg --update-trustdb
  echo '5\ny\n' | gpg --command-fd 0 --no-tty --edit-key 7D851EB72D73BDA0 trust
  gpg-connect-agent "PRESET_PASSPHRASE 3E2D1142AA59E08E16B7E2C64BA6DDC773B1A627 -1 77697468207374757069642070617373706872617365" /bye
  gpg-connect-agent "KEYINFO 3E2D1142AA59E08E16B7E2C64BA6DDC773B1A627" /bye
  gpg-connect-agent "PRESET_PASSPHRASE BA83FC8947213477F28ADC019F6564A956456163 -1 77697468207374757069642070617373706872617365" /bye
  gpg-connect-agent "KEYINFO BA83FC8947213477F28ADC019F6564A956456163" /bye
  pass init 7D851EB72D73BDA0
  gpg -k

  mkdir /out
  xx-go --wrap
  make test COVERAGEDIR=/out
EOT

FROM scratch AS test-coverage
COPY --from=test /out /

FROM gobase AS version
RUN --mount=target=. \
    echo -n "$(./hack/git-meta version)" | tee /tmp/.version ; echo -n "$(./hack/git-meta revision)" | tee /tmp/.revision

FROM base AS build-linux
ARG PACKAGE
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=/tmp/.version,target=/tmp/.version,from=version \
    --mount=type=bind,source=/tmp/.revision,target=/tmp/.revision,from=version <<EOT
  set -ex
  xx-go --wrap
  make build-pass build-secretservice PACKAGE=$PACKAGE VERSION=$(cat /tmp/.version) REVISION=$(cat /tmp/.revision) DESTDIR=/out
  xx-verify /out/docker-credential-pass
  xx-verify /out/docker-credential-secretservice
EOT

FROM base AS build-darwin
ARG PACKAGE
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,from=osxcross,src=/osxsdk,target=/xx-sdk \
    --mount=type=bind,source=/tmp/.version,target=/tmp/.version,from=version \
    --mount=type=bind,source=/tmp/.revision,target=/tmp/.revision,from=version <<EOT
  set -ex
  export MACOSX_VERSION_MIN=$(make print-MACOSX_DEPLOYMENT_TARGET)
  xx-go --wrap
  go install std
  make build-osxkeychain build-pass PACKAGE=$PACKAGE VERSION=$(cat /tmp/.version) REVISION=$(cat /tmp/.revision) DESTDIR=/out
  xx-verify /out/docker-credential-osxkeychain
  xx-verify /out/docker-credential-pass
EOT

FROM base AS build-windows
ARG PACKAGE
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=/tmp/.version,target=/tmp/.version,from=version \
    --mount=type=bind,source=/tmp/.revision,target=/tmp/.revision,from=version <<EOT
  set -ex
  xx-go --wrap
  make build-wincred PACKAGE=$PACKAGE VERSION=$(cat /tmp/.version) REVISION=$(cat /tmp/.revision) DESTDIR=/out
  mv /out/docker-credential-wincred /out/docker-credential-wincred.exe
  xx-verify /out/docker-credential-wincred.exe
EOT

FROM build-$TARGETOS AS build

FROM scratch AS binaries
COPY --from=build /out /

FROM --platform=$BUILDPLATFORM alpine AS releaser
WORKDIR /work
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN --mount=from=binaries \
    --mount=type=bind,source=/tmp/.version,target=/tmp/.version,from=version <<EOT
  set -e
  mkdir /out
  version="$(cat /tmp/.version)"
  [ "$TARGETOS" = "windows" ] && ext=".exe"
  for f in *; do
    cp "$f" "/out/${f%.*}-${version}.${TARGETOS}-${TARGETARCH}${TARGETVARIANT}${ext}"
  done
EOT

FROM scratch AS release
COPY --from=releaser /out/ /

FROM binaries
