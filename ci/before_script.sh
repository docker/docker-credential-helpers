#!/usr/bin/env bash
set -ex

if [[ "$TRAVIS_OS_NAME" == "linux"  ]]; then
    # Initialize X for `credentials` tests
    sh -e /etc/init.d/xvfb start
    sleep 3 # give xvfb some time to start
fi

# initialize GPG key for use for `pass` tests
# set temporary GNUPGHOME for key generation / access and generate a no-protection key
GNUPGHOME="$(mktemp -d .gpgtravis.XXXXXXXXXX)"
gpg --batch --generate-key <<-EOF
%echo Generating a standard key
%no-protection
Key-Type: DSA
Key-Length: 1024
Subkey-Type: ELG-E
Subkey-Length: 1024
Name-Real: Meshuggah Rocks
Name-Email: meshuggah@example.com
Expire-Date: 0
# Do a commit here, so that we can later print "done" :-)
%commit
%echo done
EOF

key=$(gpg --no-auto-check-trustdb --list-secret-keys --keyid-format LONG | grep ^sec | cut -d/ -f2 | cut -d" " -f1)
pass init $key
