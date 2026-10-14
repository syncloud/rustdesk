#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

VERSION=$1
if [[ -z "${VERSION}" ]]; then
    echo "usage $0 version"
    exit 1
fi

BUILD_DIR=${DIR}/../build/snap/rustdesk
DOWNLOAD_DIR=${DIR}/../build/download
mkdir -p ${BUILD_DIR} ${DOWNLOAD_DIR}

case $(dpkg --print-architecture) in
  amd64) ARCH=amd64 ;;
  arm64) ARCH=arm64v8 ;;
  armhf) ARCH=armv7 ;;
  *) echo "unsupported architecture"; exit 1 ;;
esac

${DIR}/../apt.sh wget unzip ca-certificates

${DIR}/../download-retry.sh \
    https://github.com/cyberb/rustdesk-server/releases/download/${VERSION}/rustdesk-server-linux-${ARCH}.zip \
    ${DOWNLOAD_DIR}/rustdesk-server.zip
unzip -o ${DOWNLOAD_DIR}/rustdesk-server.zip -d ${DOWNLOAD_DIR}

cp ${DOWNLOAD_DIR}/${ARCH}/hbbs ${BUILD_DIR}
cp ${DOWNLOAD_DIR}/${ARCH}/hbbr ${BUILD_DIR}
chmod +x ${BUILD_DIR}/hbbs ${BUILD_DIR}/hbbr
