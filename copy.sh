#!/bin/sh -e

if [ ! -e /destination ]; then
	echo Map /destination as a volume
	exit 1
fi
if [ -z ${PASSPHRASE} ]; then
	echo Set a PASSPHRASE environment variable
	exit 1
fi

gpg --batch --passphrase ${PASSPHRASE} /yard.gpg
mv yard /destination/
chmod a+x /destination/yard
