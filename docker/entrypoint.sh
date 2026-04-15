#!/bin/sh
set -eu

mkdir -p "$(dirname "$MODELRUN_DATA")" "$MODELSCOPE_CACHE"

if [ "$#" -gt 0 ]; then
  exec "$@"
fi

exec /app/modelrun
