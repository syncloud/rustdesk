#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOT=$( cd "${DIR}/.." && pwd )
BUILD_DIR=${ROOT}/build/snap

mkdir -p ${BUILD_DIR}/meta/hooks
mkdir -p ${BUILD_DIR}/bin

cd ${DIR}

go test ./... -cover

export CGO_ENABLED=0

go build -o ${BUILD_DIR}/meta/hooks/install ./cmd/install
go build -o ${BUILD_DIR}/meta/hooks/configure ./cmd/configure
go build -o ${BUILD_DIR}/meta/hooks/pre-refresh ./cmd/pre-refresh
go build -o ${BUILD_DIR}/meta/hooks/post-refresh ./cmd/post-refresh
go build -o ${BUILD_DIR}/bin/cli ./cmd/cli
