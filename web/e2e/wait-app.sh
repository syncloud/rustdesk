#!/bin/sh -e

APP_DOMAIN=$1

if [ -z "$APP_DOMAIN" ]; then
    echo "usage $0 app_domain"
    exit 1
fi

for i in $(seq 1 120); do
  code=$(curl -k -s -o /dev/null -w '%{http_code}' "https://${APP_DOMAIN}" || true)
  if [ "$code" = "200" ]; then
    echo "${APP_DOMAIN} is up"
    exit 0
  fi
  echo "waiting for ${APP_DOMAIN}, got '${code}' (${i}/120)"
  sleep 5
done

echo "${APP_DOMAIN} did not become available"
exit 1
