# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23.6
ARG DISTRO=ubuntu
ARG SUITE=focal

FROM golang:${GO_VERSION}-bullseye AS golang

FROM ${DISTRO}:${SUITE}

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -yy debhelper dh-make libsecret-1-dev
RUN mkdir -p /build

WORKDIR /build
ENV GOPROXY=https://proxy.golang.org|direct
ENV GO111MODULE=off
ENV GOPATH=/build
ENV PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
COPY --from=golang /usr/local/go /usr/local/go

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
