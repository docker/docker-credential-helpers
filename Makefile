.PHONY: all deps osxkeychain test validate wincred

all: test

deps:
	go get github.com/golang/lint/golint

osxkeychain:
	mkdir -p bin
	go build -o bin/docker-credential-osxkeychain osxkeychain/cmd/main_darwin.go

test:
	# tests all packages except vendor
	go test -v `go list ./... | grep -v /vendor/`

validate:
	go vet ./credentials ./osxkeychain
	golint `go list ./... | grep -v /vendor/`
	gofmt -s -l `ls **/*.go | grep -v vendor`

wincred:
	mkdir -p bin
	go build -o bin/docker-credential-wincred wincred/cmd/main_windows.go
