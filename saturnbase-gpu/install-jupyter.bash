#!/bin/bash

set -ex

cd $(dirname $0)

echo "installing root env:"
cat /tmp/environment.yml
conda env update -n root  -f /tmp/environment.yml
${CONDA_DIR}/bin/jupyter serverextension enable --py nbserverproxy --sys-prefix

${CONDA_DIR}/bin/jupyter labextension install jupyterlab_bokeh@1.0.0
${CONDA_DIR}/bin/jupyter labextension install @jupyter-widgets/jupyterlab-manager

cd ${CONDA_DIR}/jsaturn_ext
${CONDA_DIR}/bin/npm install
${CONDA_DIR}/bin/npm run build
${CONDA_DIR}/bin/jupyter labextension install

cd ${HOME}
rm -rf ${CONDA_DIR}/jsaturn_ext

${CONDA_DIR}/bin/conda clean -afy
${CONDA_DIR}/bin/jupyter lab clean
${CONDA_DIR}/bin/jlpm cache clean
${CONDA_DIR}/bin/npm cache clean --force
find ${CONDA_DIR} -type f,l -name '*.pyc' -delete
find ${CONDA_DIR} -type f,l -name '*.a' -delete
find ${CONDA_DIR} -type f,l -name '*.js.map' -delete
rm -rf $HOME/.node-gyp
rm -rf $HOME/.local

${CONDA_DIR}/bin/conda create -n saturn
