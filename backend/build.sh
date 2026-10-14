#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOT=$( cd "${DIR}/.." && pwd )
BUILD_DIR=${ROOT}/build/snap/backend

mkdir -p ${BUILD_DIR}

cd ${DIR}

go test ./... -cover

export CGO_ENABLED=0

go build -trimpath -ldflags '-s -w' -o ${BUILD_DIR}/backend ./cmd/backend
