#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOT=$( cd "${DIR}/.." && pwd )
BUILD_DIR=${ROOT}/build/snap

${BUILD_DIR}/bin/cli --help
${BUILD_DIR}/meta/hooks/install --help
${BUILD_DIR}/meta/hooks/configure --help
${BUILD_DIR}/meta/hooks/pre-refresh --help
${BUILD_DIR}/meta/hooks/post-refresh --help
