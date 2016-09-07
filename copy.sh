#!/bin/sh -e

if [ ! -e /destination ]; then
	echo Map /destination as a volume
	exit 1
fi

cp -f gluon /destination/
chmod a+x /destination/gluon

mkdir -p /destination/overlay
mkdir -p /destination/overlay-work

cp -f /etcd* /destination/overlay/
chmod a+x /destination/overlay/etcd*

cp -f /fleet* /destination/overlay/
chmod a+x /destination/overlay/fleet*

cp -f /rkt /destination/overlay/
cp -f /stage1-coreos.aci /destination/overlay/
chmod a+x /destination/overlay/rkt
