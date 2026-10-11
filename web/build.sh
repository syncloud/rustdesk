#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOT=$( cd "${DIR}/.." && pwd )
BUILD_DIR=${ROOT}/build/snap/web/dist

cd ${DIR}

${ROOT}/npm.sh ci --no-audit --no-fund
npm run build

mkdir -p ${BUILD_DIR}
cp -r dist/* ${BUILD_DIR}
