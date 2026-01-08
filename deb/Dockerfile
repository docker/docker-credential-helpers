# syntax=docker/dockerfile:1

# GO_VERSION sets the version of the golang base image to use.
# It must be a valid tag in the docker.io/library/golang image repository.
ARG GO_VERSION=1.26.3

# BASE_DEBIAN_DISTRO sets the golang base image debian variant to use.
# It must be a valid variant in the docker.io/library/golang image repository.
ARG BASE_DEBIAN_DISTRO=trixie

ARG DISTRO=ubuntu
ARG SUITE=jammy

FROM golang:${GO_VERSION}-${BASE_DEBIAN_DISTRO} AS gobase

FROM ${DISTRO}:${SUITE}
RUN apt-get update && apt-get install -yy debhelper dh-make libsecret-1-dev
RUN mkdir -p /build

WORKDIR /build
ENV GOPROXY=https://proxy.golang.org|direct
ENV GO111MODULE=off
ENV GOPATH=/build
ENV PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
COPY --from=gobase /usr/local/go /usr/local/go

COPY Makefile .
COPY credentials credentials
COPY secretservice secretservice
COPY pass pass
COPY deb/debian ./debian
COPY deb/build-deb .

ARG VERSION
ENV VERSION=${VERSION}
ARG REVISION
ENV REVISION=${REVISION}
ARG DISTRO
ENV DISTRO=${DISTRO}
ARG SUITE
ENV SUITE=${SUITE}
RUN /build/build-deb
