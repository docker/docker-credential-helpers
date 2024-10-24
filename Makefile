PACKAGE ?= github.com/docker/docker-credential-helpers
VERSION ?= $(shell ./hack/git-meta version)
REVISION ?= $(shell ./hack/git-meta revision)

GO_PKG = github.com/docker/docker-credential-helpers
GO_LDFLAGS = -s -w -X ${GO_PKG}/credentials.Version=${VERSION} -X ${GO_PKG}/credentials.Revision=${REVISION} -X ${GO_PKG}/credentials.Package=${PACKAGE}

BUILDX_CMD ?= docker buildx
DESTDIR ?= ./bin/build
COVERAGEDIR ?= ./bin/coverage

# 10.11 is the minimum supported version for osxkeychain
export MACOSX_DEPLOYMENT_TARGET = 10.11
ifeq "$(shell go env GOOS)" "darwin"
	export CGO_CFLAGS = -mmacosx-version-min=$(MACOSX_DEPLOYMENT_TARGET)
endif

.PHONY: all
all: cross

.PHONY: clean
clean:
	rm -rf bin

.PHONY: build-%
build-%: # build, can be one of build-osxkeychain build-pass build-secretservice build-wincred
	go build -trimpath -ldflags="$(GO_LDFLAGS) -X ${GO_PKG}/credentials.Name=docker-credential-$*" -o "$(DESTDIR)/docker-credential-$*" ./$*/cmd/

# aliases for build-* targets
.PHONY: osxkeychain secretservice pass wincred keyctl
osxkeychain: build-osxkeychain
secretservice: build-secretservice
pass: build-pass
wincred: build-wincred
keyctl: build-keyctl

.PHONY: cross
cross: # cross build all supported credential helpers
	$(BUILDX_CMD) bake binaries

.PHONY: release
release: # create release
	./hack/release

.PHONY: test
test:
	mkdir -p $(COVERAGEDIR)
	go test -short -v -coverprofile=$(COVERAGEDIR)/coverage.txt -covermode=atomic ./...
	go tool cover -func=$(COVERAGEDIR)/coverage.txt

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
		--build-arg VERSION=$(patsubst v%,%,$(VERSION)) \
		--build-arg REVISION=$(REVISION) \
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

.PHONY: print-%
print-%: ; @echo $($*)
