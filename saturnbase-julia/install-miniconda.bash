#!/bin/bash
cd /tmp

set -x && \
    MINICONDA_URL="https://repo.anaconda.com/miniconda/Miniconda3-${CONDA_VERSION}-Linux-x86_64.sh"; \
    SHA256SUM="4ee9c3aa53329cd7a63b49877c0babb49b19b7e5af29807b793a76bdb1d362b4"; \
    wget "${MINICONDA_URL}" -O miniconda.sh -q && \
    echo "${SHA256SUM} miniconda.sh" > shasum && \
    if [ "${CONDA_VERSION}" != "latest" ]; then sha256sum --check --status shasum; fi && \
    mkdir -p /opt && \
    sh miniconda.sh -b -p /opt/saturncloud && \
    rm miniconda.sh shasum && \
    ln -s /opt/saturncloud/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    echo ". /opt/saturncloud/etc/profile.d/conda.sh" >> ~/.bashrc && \
    echo "conda activate base" >> ~/.bashrc && \
    find /opt/saturncloud/ -follow -type f -name '*.a' -delete && \
    find /opt/saturncloud/ -follow -type f -name '*.js.map' -delete && \
    /opt/saturncloud/bin/conda clean -afy && \
    chown -R 1000:1000 /opt/saturncloud
/opt/saturncloud/bin/conda install conda=4.13
/opt/saturncloud/bin/conda install -c conda-forge mamba=0.25
