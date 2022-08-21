# syntax=docker/dockerfile:1

ARG GO_VERSION=1.18.5
ARG XX_VERSION=1.1.2
ARG OSXCROSS_VERSION=11.3-r7-alpine

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
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
  set -e
  xx-go test -short -v -coverprofile=/tmp/coverage.txt -covermode=atomic ./...
  xx-go tool cover -func=/tmp/coverage.txt
EOT

FROM scratch AS test-coverage
COPY --from=test /tmp/coverage.txt /coverage.txt

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
