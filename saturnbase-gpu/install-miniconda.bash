#!/bin/bash

set -ex

cd $(dirname $0)

MINICONDA_VERSION=py38_4.8.2
URL="https://repo.continuum.io/miniconda/Miniconda3-${MINICONDA_VERSION}-Linux-x86_64.sh"
INSTALLER_PATH=/tmp/miniconda-installer.sh
wget --quiet $URL -O ${INSTALLER_PATH}
chmod +x ${INSTALLER_PATH}

MD5SUM="cbda751e713b5a95f187ae70b509403f"

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
