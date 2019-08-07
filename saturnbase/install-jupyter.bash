#!/bin/bash

set -ex

cd $(dirname $0)

echo "installing root env:"
cat /tmp/jupyter.yml
conda env update -n root  -f /tmp/jupyter.yml
echo '' > ${CONDA_DIR}/conda-meta/history
