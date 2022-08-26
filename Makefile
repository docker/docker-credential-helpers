PACKAGE ?= github.com/docker/docker-credential-helpers
VERSION ?= $(shell git describe --match 'v[0-9]*' --dirty='.m' --always --tags)
REVISION ?= $(shell git rev-parse HEAD)$(shell if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi)

GO_PKG = github.com/docker/docker-credential-helpers
GO_LDFLAGS = -s -w -X ${GO_PKG}/credentials.Version=${VERSION} -X ${GO_PKG}/credentials.Revision=${REVISION} -X ${GO_PKG}/credentials.Package=${PACKAGE}

BUILDX_CMD ?= docker buildx
DESTDIR ?= ./bin/build

.PHONY: all
all: cross

.PHONY: clean
clean:
	rm -rf bin

.PHONY: build-%
build-%: # build, can be one of build-osxkeychain build-pass build-secretservice build-wincred
	$(eval BINNAME := docker-credential-$*)
	go build -trimpath -ldflags="$(GO_LDFLAGS) -X ${GO_PKG}/credentials.Name=docker-credential-$*" -o $(DESTDIR)/$(BINNAME) ./$*/cmd/

# aliases for build-* targets
.PHONY: osxkeychain secretservice pass wincred
osxkeychain: build-osxkeychain
secretservice: build-secretservice
pass: build-pass
wincred: build-wincred

.PHONY: osxcodesign
osxcodesign: build-osxkeychain
	$(eval SIGNINGHASH = $(shell security find-identity -v -p codesigning | grep "Developer ID Application: Docker Inc" | cut -d ' ' -f 4))
	xcrun -log codesign -s $(SIGNINGHASH) --force --verbose bin/build/docker-credential-osxkeychain
	xcrun codesign --verify --deep --strict --verbose=2 --display bin/build/docker-credential-osxkeychain

.PHONY: linuxrelease
linuxrelease:
	mkdir -p release
	cd bin && tar cvfz ../release/docker-credential-pass-$(VERSION)-amd64.tar.gz docker-credential-pass
	cd bin && tar cvfz ../release/docker-credential-secretservice-$(VERSION)-amd64.tar.gz docker-credential-secretservice

.PHONY: osxrelease
osxrelease:
	mkdir -p release
	cd bin && tar cvfz ../release/docker-credential-osxkeychain-$(VERSION)-amd64.tar.gz docker-credential-osxkeychain
	cd bin && tar cvfz ../release/docker-credential-pass-$(VERSION)-darwin-amd64.tar.gz docker-credential-pass

.PHONY: winrelease
winrelease:
	mkdir -p release
	cd bin && zip ../release/docker-credential-wincred-$(VERSION)-amd64.zip docker-credential-wincred.exe

.PHONY: cross
cross: # cross build all supported credential helpers
	$(BUILDX_CMD) bake cross

.PHONY: test
test:
	# tests all packages except vendor
	go test -v `go list ./... | grep -v /vendor/`

.PHONY: lint
lint:
	$(BUILDX_CMD) bake lint

.PHONY: validate-vendor
validate-vendor:
	$(BUILDX_CMD) bake vendor-validate

.PHONY: fmt
fmt:
	gofmt -s -l `ls **/*.go | grep -v vendor`

.PHONY: validate
validate: lint validate-vendor fmt

BUILDIMG:=docker-credential-secretservice-$(VERSION)
.PHONY: deb
deb:
	mkdir -p release
	docker build -f deb/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg DISTRO=xenial \
		--tag $(BUILDIMG) \
		.
	docker run --rm --net=none $(BUILDIMG) tar cf - /release | tar xf -
	docker rmi $(BUILDIMG)

.PHONY: vendor
vendor:
	$(eval $@_TMP_OUT := $(shell mktemp -d -t docker-output.XXXXXXXXXX))
	$(BUILDX_CMD) bake --set "*.output=type=local,dest=$($@_TMP_OUT)" vendor
	rm -rf ./vendor
	cp -R "$($@_TMP_OUT)"/* .
	rm -rf "$($@_TMP_OUT)"
