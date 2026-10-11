#!/bin/bash -e

umask 077
cd $SNAP_DATA/server
exec $SNAP/rustdesk/hbbs -k _ --ws-socket $SNAP_DATA/hbbs.sock --ws-only Y
