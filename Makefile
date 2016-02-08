.PHONY: all deps osxkeychain test

all: test

deps:
	go get -t ./...
	go get github.com/golang/lint/golint

osxkeychain:
	mkdir -p bin
	go build -o bin/docker-credential-osxkeychain osxkeychain/cmd/main_darwin.go

test:
	go test -v ./...

validate:
	go vet ./...
	golint ./...
	gofmt -s -l .
