#!/bin/sh -e

if [ ! -e /destination ]; then
	echo Map /destination as a volume
	exit 1
fi

cp -f /dist/gluon /destination/
chmod a+x /destination/gluon

mkdir -p /destination/overlay
mkdir -p /destination/overlay-work

cp -f /dist/etcd* /destination/overlay/
chmod a+x /destination/overlay/etcd*

cp -f /dist/fleet* /destination/overlay/
chmod a+x /destination/overlay/fleet*

cp -f /dist/rkt/rkt /destination/overlay/
cp -f /dist/rkt/stage1*.aci /destination/overlay/
mkdir -p /destination/overlay/rkt-scripts
cp -f /dist/rkt/setup-data-dir.sh /destination/overlay/rkt-scripts/
chmod a+x /destination/overlay/rkt
