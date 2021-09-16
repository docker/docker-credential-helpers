# syntax=docker/dockerfile:1.3
ARG GO_VERSION=1.16

FROM golang:${GO_VERSION}-alpine
RUN apk add --no-cache gcc libsecret-dev musl-dev
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.36.0
WORKDIR /go/src/github.com/docker/buildx
RUN --mount=target=/go/src/github.com/docker/buildx --mount=target=/root/.cache,type=cache \
  golangci-lint run
