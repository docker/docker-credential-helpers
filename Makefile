.PHONY: all osxkeychain secretservice test lint validate-vendor fmt validate wincred pass deb vendor

VERSION := $(shell grep 'const Version' credentials/version.go | awk -F'"' '{ print $$2 }')

all: test

clean:
	rm -rf bin
	rm -rf release

osxkeychain:
	mkdir -p bin
	go build -ldflags -s -o bin/docker-credential-osxkeychain ./osxkeychain/cmd/

osxcodesign: osxkeychain
	$(eval SIGNINGHASH = $(shell security find-identity -v -p codesigning | grep "Developer ID Application: Docker Inc" | cut -d ' ' -f 4))
	xcrun -log codesign -s $(SIGNINGHASH) --force --verbose bin/docker-credential-osxkeychain
	xcrun codesign --verify --deep --strict --verbose=2 --display bin/docker-credential-osxkeychain

secretservice:
	mkdir -p bin
	go build -o bin/docker-credential-secretservice ./secretservice/cmd/

pass:
	mkdir -p bin
	go build -o bin/docker-credential-pass ./pass/cmd/

wincred:
	mkdir -p bin
	go build -o bin/docker-credential-wincred.exe ./wincred/cmd/

linuxrelease:
	mkdir -p release
	cd bin && tar cvfz ../release/docker-credential-pass-v$(VERSION)-amd64.tar.gz docker-credential-pass
	cd bin && tar cvfz ../release/docker-credential-secretservice-v$(VERSION)-amd64.tar.gz docker-credential-secretservice

osxrelease:
	mkdir -p release
	cd bin && tar cvfz ../release/docker-credential-osxkeychain-v$(VERSION)-amd64.tar.gz docker-credential-osxkeychain
	cd bin && tar cvfz ../release/docker-credential-pass-v$(VERSION)-darwin-amd64.tar.gz docker-credential-pass

winrelease:
	mkdir -p release
	cd bin && zip ../release/docker-credential-wincred-v$(VERSION)-amd64.zip docker-credential-wincred.exe

test:
	# tests all packages except vendor
	go test -v `go list ./... | grep -v /vendor/`

lint:
	docker buildx bake lint

validate-vendor:
	docker buildx bake vendor-validate

fmt:
	gofmt -s -l `ls **/*.go | grep -v vendor`

validate: lint validate-vendor fmt

BUILDIMG:=docker-credential-secretservice-$(VERSION)
deb:
	mkdir -p release
	docker build -f deb/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg DISTRO=xenial \
		--tag $(BUILDIMG) \
		.
	docker run --rm --net=none $(BUILDIMG) tar cf - /release | tar xf -
	docker rmi $(BUILDIMG)

vendor:
	$(eval $@_TMP_OUT := $(shell mktemp -d -t docker-output.XXXXXXXXXX))
	docker buildx bake --set "*.output=type=local,dest=$($@_TMP_OUT)" vendor
	rm -rf ./vendor
	cp -R "$($@_TMP_OUT)"/* .
	rm -rf "$($@_TMP_OUT)"
