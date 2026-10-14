#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}/../test

SPEC=$1
DISTRO=$2
APP=$3

if [ -z "$APP" ]; then
    echo "usage $0 spec distro app"
    exit 1
fi

./deps.sh
py.test -x -s ${SPEC} --distro=${DISTRO} --ver=${DRONE_BUILD_NUMBER} --app=${APP}
