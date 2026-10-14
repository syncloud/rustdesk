#!/bin/bash -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

for i in $(seq 1 10); do
  if npm "$@"; then
    exit 0
  fi
  echo "retry npm $*"
  sleep 10
done

echo "npm $* failed"
exit 1
