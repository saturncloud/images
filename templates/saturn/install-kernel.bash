#!/bin/bash

set -ex

conda env update -n saturn -f environment.yml
echo '' > ${CONDA_DIR}/conda-meta/history
${CONDA_DIR}/bin/conda clean --yes --all
