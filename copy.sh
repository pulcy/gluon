#!/bin/sh -e

if [ ! -e /destination ]; then
	echo Map /destination as a volume
	exit 1
fi

cp -f gluon /destination/
chmod a+x /destination/gluon
