# syntax=docker/dockerfile:1.3-labs
ARG GO_VERSION=1.16

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.0.0-rc.2 AS xx

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS gobase
COPY --from=xx / /
RUN apk add --no-cache clang file gcc git libsecret-dev lld musl-dev pass
ENV GOFLAGS="-mod=vendor"
ENV CGO_ENABLED="1"
WORKDIR /src

FROM gobase AS version
RUN --mount=target=. \
  PKG=github.com/docker/docker-credential-helpers VERSION=$(git describe --match 'v[0-9]*' --dirty='.m' --always --tags) REVISION=$(git rev-parse HEAD)$(if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi); \
  echo "-s -w -X ${PKG}/credentials.Version=${VERSION} -X ${PKG}/credentials.Revision=${REVISION} -X ${PKG}/credentials.Package=${PKG}" | tee /tmp/.ldflags; \
  echo -n "${VERSION}" | tee /tmp/.version;

FROM gobase AS build-linux
ARG TARGETOS
ARG TARGETPLATFORM
RUN --mount=type=bind,target=. \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=bind,from=version,source=/tmp/.ldflags,target=/tmp/.ldflags <<EOT
set -ex
mkdir /out
xx-go build -ldflags "$(cat /tmp/.ldflags)" -o /out/docker-credential-pass ./pass/cmd/main.go
xx-verify /out/docker-credential-pass
xx-go build -ldflags "$(cat /tmp/.ldflags)" -o /out/docker-credential-secretservice ./secretservice/cmd/main_linux.go
xx-verify /out/docker-credential-secretservice
EOT

FROM gobase AS build-darwin
ARG TARGETOS
ARG TARGETPLATFORM
RUN --mount=type=bind,target=. \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=bind,from=dockercore/golang-cross:xx-sdk-extras,src=/xx-sdk,target=/xx-sdk \
  --mount=type=bind,from=version,source=/tmp/.ldflags,target=/tmp/.ldflags <<EOT
set -ex
mkdir /out
xx-go install std
xx-go build -ldflags "$(cat /tmp/.ldflags)" -o /out/docker-credential-osxkeychain ./osxkeychain/cmd/main_darwin.go
xx-verify /out/docker-credential-osxkeychain
EOT

FROM gobase AS build-windows
ARG TARGETOS
ARG TARGETPLATFORM
RUN --mount=type=bind,target=. \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=bind,from=version,source=/tmp/.ldflags,target=/tmp/.ldflags <<EOT
set -ex
mkdir /out
xx-go build -ldflags "$(cat /tmp/.ldflags)" -o /out/docker-credential-wincred.exe ./wincred/cmd/main_windows.go
xx-verify /out/docker-credential-wincred.exe
EOT

FROM build-$TARGETOS AS build

FROM scratch AS binaries
COPY --from=build /out /

FROM debian:bullseye-slim AS build-deb
RUN apt-get update && \
  apt-get install -y debhelper dh-make libsecret-1-dev pass
WORKDIR /build
COPY --from=build /out/docker-credential-pass ./docker-credential-pass/usr/bin/
COPY --from=build /out/docker-credential-secretservice ./docker-credential-secretservice/usr/bin/
RUN --mount=type=bind,from=version,source=/tmp/.version,target=/tmp/.version <<EOT
#!/usr/bin/env bash
set -e
version=$(cat /tmp/.version)
if [ ${#version} = 7 ]; then
  version="v0.0.0+${version}"
fi
mkdir -p ./docker-credential-pass/DEBIAN
cat > ./docker-credential-pass/DEBIAN/control <<EOL
Package: docker-credential-pass
Version: ${version:1}
Architecture: any
Depends: pass
Maintainer: Docker <support@docker.com>
Description: docker-credential-pass is a credential helper backend
 which uses the pass utility to keep Docker credentials safe.
EOL
mkdir -p ./docker-credential-secretservice/DEBIAN
cat > ./docker-credential-secretservice/DEBIAN/control <<EOL
Package: docker-credential-secretservice
Version: ${version:1}
Architecture: any
Depends: libsecret-1-0
Maintainer: Docker <support@docker.com>
Description: docker-credential-secretservice is a credential helper backend
 which uses libsecret to keep Docker credentials safe.
EOL
dpkg-deb --build docker-credential-pass
dpkg-deb --build docker-credential-secretservice
EOT

FROM scratch AS deb
COPY --from=build-deb /build/*.deb /

FROM gobase AS test
RUN --mount=type=bind,target=. \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod <<EOT
set -e
xx-go test -short -v -coverprofile=/tmp/coverage.txt -covermode=atomic ./...
xx-go tool cover -func=/tmp/coverage.txt
EOT

FROM scratch AS test-coverage
COPY --from=test /tmp/coverage.txt /coverage.txt

FROM binaries
