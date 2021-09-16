buildxCmd =
ifneq (, $(BUILDX_PATH))
	buildxCmd = $(BUILDX_PATH)
else ifneq (, $(shell docker buildx version))
	buildxCmd = docker buildx
else ifneq (, $(shell which buildx))
	buildxCmd = $(which buildx)
else
	$(error "Please install Buildx: https://github.com/docker/buildx#installing")
endif

BIN_OUT = ./bin
RELEASE_OUT = ./release

binaries:
	rm -rf $(BIN_OUT)
	BIN_OUT=$(BIN_OUT) $(buildxCmd) bake binaries

deb:
	BIN_OUT=$(BIN_OUT) $(buildxCmd) bake deb

release: binaries
	rm -rf $(RELEASE_OUT)
	mkdir -p $(RELEASE_OUT)
	RELEASE_OUT=$(RELEASE_OUT) ./hack/release

validate-all: lint test vendor-validate

lint:
	$(buildxCmd) bake lint

test:
	$(buildxCmd) bake test

vendor-validate:
	$(buildxCmd) bake vendor-validate

vendor:
	$(eval $@_TMP_OUT := $(shell mktemp -d -t buildx-output.XXXXXXXXXX))
	$(buildxCmd) bake --set "*.output=$($@_TMP_OUT)" vendor-update
	rm -rf ./vendor
	cp -R "$($@_TMP_OUT)"/out/* .
	rm -rf $($@_TMP_OUT)/*

.PHONY: clean binaries deb release validate-all lint test vendor-validate vendor
