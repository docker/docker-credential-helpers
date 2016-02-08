.PHONY: all osxkeychain test

all: test

osxkeychain:
	mkdir -p bin
	go build -o bin/docker-credential-osxkeychain osxkeychain/cmd/main_darwin.go

test:
	go test ./...
