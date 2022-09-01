#!/bin/bash

set -ex
export PATH="${CONDA_BIN}:${PATH}"

cd $(dirname $0)

echo "installing root env:"
cat /tmp/environment.yml
mamba env update -n root  -f /tmp/environment.yml

conda clean -afy
find ${CONDA_DIR}/ -type f,l -name '*.pyc' -delete
find ${CONDA_DIR}/ -type f,l -name '*.a' -delete
find ${CONDA_DIR}/ -type f,l -name '*.js.map' -delete

conda create -n saturn
