#!/usr/bin/env sh
set -ex

# shellcheck disable=SC2155
export GPG_TTY=$(tty)
gpg --batch --gen-key --no-tty <<-EOF
%echo Generating a standard key
Key-Type: DSA
Key-Length: 1024
Subkey-Type: ELG-E
Subkey-Length: 1024
Name-Real: Meshuggah Rocks
Name-Email: meshuggah@example.com
Passphrase: with stupid passphrase
Expire-Date: 0
# Do a commit here, so that we can later print "done" :-)
%commit
%echo done
EOF

# doesn't work; still asks for passphrase interactively
# gpg --output private.pgp --armor --export-secret-key meshuggah@example.com

# doesn't work; still asks for passphrase interactively
# gpg --passphrase 'with stupid passphrase' --output private.pgp --armor --export-secret-key meshuggah@example.com

# doesn't work; still asks for passphrase interactively
# gpg --batch --passphrase 'with stupid passphrase' --no-tty --output private.pgp --armor --export-secret-key meshuggah@example.com
