#!/bin/bash

set -ex

cd $(dirname $0)

MINICONDA_VERSION=4.5.11
CONDA_VERSION=4.7.5
# CONDA_VERSION=4.5.11
URL="https://repo.continuum.io/miniconda/Miniconda3-${MINICONDA_VERSION}-Linux-x86_64.sh"
INSTALLER_PATH=/tmp/miniconda-installer.sh
wget --quiet $URL -O ${INSTALLER_PATH}
chmod +x ${INSTALLER_PATH}

MD5SUM="e1045ee415162f944b6aebfe560b8fee"

if ! echo "${MD5SUM}  ${INSTALLER_PATH}" | md5sum  --quiet -c -; then
    echo "md5sum mismatch for ${INSTALLER_PATH}, exiting!"
    exit 1
fi

bash ${INSTALLER_PATH} -b -p ${CONDA_DIR}
export PATH="${CONDA_DIR}/bin:$PATH"

# Allow easy direct installs from conda forge
conda config --system --add channels conda-forge
conda config --system --add channels https://conda.saturncloud.io/pkgs
conda config --system --set auto_update_conda false
conda config --system --set show_channel_urls true
conda install -yq conda==${CONDA_VERSION}
