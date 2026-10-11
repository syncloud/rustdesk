#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOT=$( cd "${DIR}/.." && pwd )

PROJECT=$1
APP=$2
DISTRO=$3
SPEC=$4

if [ -z "$SPEC" ]; then
    echo "usage $0 project app distro spec"
    exit 1
fi

getent hosts ${APP}.${DISTRO}.com | sed "s/${APP}.${DISTRO}.com/auth.${DISTRO}.com/g" | tee -a /etc/hosts

ARTIFACT=${ROOT}/artifact/screenshots
mkdir -p ${ARTIFACT}
trap 'chmod -R a+r ${ARTIFACT} 2>/dev/null; exit' EXIT INT TERM

cd ${ROOT}/web
./e2e/wait-app.sh ${APP}.${DISTRO}.com
${ROOT}/npm.sh ci --no-audit --no-fund

PLAYWRIGHT_DOMAIN=${DISTRO}.com \
PLAYWRIGHT_APP=${APP} \
PLAYWRIGHT_ARTIFACT_DIR=${ARTIFACT} \
  npx playwright test --project=${PROJECT} "${SPEC}"
