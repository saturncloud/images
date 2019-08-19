#!/bin/bash

set -ex
ls $CONDA_DIR/envs
source activate saturn
python -m ipykernel install --name python2 --prefix=/srv/conda