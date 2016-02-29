set -ex

mkdir bin
go build -o bin/docker-credential-osxkeychain osxkeychain/cmd/main_darwin.go
cd bin 
tar czf ../docker-credential-osxkeychain-${TRAVIS_TAG}-amd64.tar.gz docker-credential-osxkeychain
