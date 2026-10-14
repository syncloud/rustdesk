#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOT=$( cd "${DIR}/.." && pwd )

ROLE=$1
VERSION=$2
DOMAIN=$3

if [ -z "$DOMAIN" ]; then
    echo "usage $0 workstation|control client_version app_domain"
    exit 1
fi

export DEBIAN_FRONTEND=noninteractive
${ROOT}/apt.sh --no-install-recommends ca-certificates wget xvfb x11-utils imagemagick openbox procps xdg-user-dirs \
    libayatana-appindicator3-1 libegl1 libgl1 libgles2 libgl1-mesa-dri python3-pytest \
    xfce4 xfce4-terminal dbus-x11 adwaita-icon-theme fonts-dejavu-core
${ROOT}/download-retry.sh \
    https://github.com/rustdesk/rustdesk/releases/download/${VERSION}/rustdesk-${VERSION}-x86_64.deb \
    /tmp/rustdesk.deb
${ROOT}/apt.sh --no-install-recommends /tmp/rustdesk.deb

xdg-user-dirs-update
Xvfb :0 -screen 0 1280x800x24 &
export DISPLAY=:0

cd ${ROOT}/test/client
python3 -m pytest -x -s ${ROLE}.py --role=${ROLE} --domain=${DOMAIN}
