#!/bin/bash

set -ex

cd $(dirname $0)

MINICONDA_VERSION=py37_4.8.2
URL="https://repo.continuum.io/miniconda/Miniconda3-${MINICONDA_VERSION}-Linux-x86_64.sh"
INSTALLER_PATH=/tmp/miniconda-installer.sh
wget --quiet $URL -O ${INSTALLER_PATH}
chmod +x ${INSTALLER_PATH}

MD5SUM="87e77f097f6ebb5127c77662dfc3165e"

if ! echo "${MD5SUM}  ${INSTALLER_PATH}" | md5sum  --quiet -c -; then
    echo "md5sum mismatch for ${INSTALLER_PATH}, exiting!"
    exit 1
fi

bash ${INSTALLER_PATH} -b -p ${CONDA_DIR}

# Allow easy direct installs from conda forge
${CONDA_BIN}/conda config --system --add channels conda-forge
${CONDA_BIN}/conda config --system --add channels https://conda.saturncloud.io/pkgs
${CONDA_BIN}/conda config --system --set auto_update_conda false
${CONDA_BIN}/conda config --system --set show_channel_urls true
