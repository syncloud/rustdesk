#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap/rustdesk
${BUILD_DIR}/hbbs --version
${BUILD_DIR}/hbbr --version
