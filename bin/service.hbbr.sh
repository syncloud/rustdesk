#!/bin/bash -e

umask 077
cd $SNAP_DATA/server

while [ ! -f id_ed25519.pub ]
do
  echo "waiting for hbbs to generate id_ed25519.pub"
  sleep 1
done

exec $SNAP/rustdesk/hbbr -k _ --ws-socket $SNAP_DATA/hbbr.sock --ws-only Y
