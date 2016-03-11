#!/bin/bash
set -e

cd /
#echo $@
echo "Extracting base64 encoded files"
echo $DATA | base64 -d | tar xv
exec "$@"
