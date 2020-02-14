#!/bin/bash

set -ex

cd $(dirname $0)

echo "installing root env:"
cat /tmp/environment.yml
conda env update -n root  -f /tmp/environment.yml
echo '' > ${CONDA_DIR}/conda-meta/history
