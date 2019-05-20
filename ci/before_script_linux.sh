set -ex
ARCH_TYPE=$( arch )
if [ $ARCH_TYPE = "ppc64le" ]; then
	sudo service xvfb start
else
	sh -e /etc/init.d/xvfb start
fi
sleep 3 # give xvfb some time to start

# init key for pass
gpg --batch --gen-key <<-EOF
%echo Generating a standard key
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

key=$(gpg --no-auto-check-trustdb --list-secret-keys | grep ^sec | cut -d/ -f2 | cut -d" " -f1)
pass init $key
