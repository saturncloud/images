#!/bin/bash

set -ex
export PATH="${CONDA_BIN}:${PATH}"

cd $(dirname $0)

echo "installing root env:"
cat /tmp/environment.yml
conda install -c conda-forge mamba
mamba env update -n root  -f /tmp/environment.yml

conda clean -afy

conda create -n saturn