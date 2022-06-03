#!/bin/bash

set -ex

cd $(dirname $0)

MINICONDA_VERSION=py38_4.9.2
URL="https://repo.continuum.io/miniconda/Miniconda3-${MINICONDA_VERSION}-Linux-x86_64.sh"
INSTALLER_PATH=/tmp/miniconda-installer.sh
wget --quiet $URL -O ${INSTALLER_PATH}
chmod +x ${INSTALLER_PATH}

MD5SUM="122c8c9beb51e124ab32a0fa6426c656"

if ! echo "${MD5SUM}  ${INSTALLER_PATH}" | md5sum  --quiet -c -; then
    echo "md5sum mismatch for ${INSTALLER_PATH}, exiting!"
    exit 1
fi

bash ${INSTALLER_PATH} -b -p ${CONDA_DIR} -f

export PATH="${CONDA_BIN}:$PATH"

# Update conda
conda install -y conda=4.12


# Allow easy direct installs from conda forge
conda config --system --add channels conda-forge
conda config --system --set auto_update_conda false
conda config --system --set show_channel_urls true
