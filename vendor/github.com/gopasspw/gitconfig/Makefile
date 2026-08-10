GO                        ?= GO111MODULE=on CGO_ENABLED=0 go
GOFILES_NOVENDOR          := $(shell find . -name vendor -prune -o -type f -name '*.go' -not -name '*.pb.go' -print)
OK := $(shell tput setaf 6; echo ' [OK]'; tput sgr0;)

crosscompile:
	@echo ">> CROSS-COMPILE"

	@echo -n "     BUILDING FOR LINUX AMD64 "
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o bin/gocopy-linux-amd64 ./cmd/gocopy || exit 1
	@printf '%s\n' '$(OK)'

	@echo -n "     BUILDING FOR WINDOWS AMD64 "
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o bin/gocopy-linux-amd64 ./cmd/gocopy || exit 1
	@printf '%s\n' '$(OK)'

	@echo -n "     BUILDING FOR MACOS AMD64 "
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o bin/gocopy-linux-amd64 ./cmd/gocopy || exit 1
	@printf '%s\n' '$(OK)'

codequality:
	@echo ">> CODE QUALITY"

	@echo -n "     GOLANGCI-LINT "
	@which golangci-lint > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.1; \
	fi
	@golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 || exit 1

	@printf '%s\n' '$(OK)'

test:
	@echo ">> TEST"

	@echo -n "     UNIT TESTS "
	@$(GO) test -v

fmt:
	@keep-sorted --mode fix $(GOFILES_NOVENDOR)
	@gofumpt -w $(GOFILES_NOVENDOR)
	@$(GO) mod tidy


.PHONY: clean build crosscompile test codequality
