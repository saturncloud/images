#!/bin/bash
cd /tmp

set -x && \
    MINICONDA_URL="https://repo.anaconda.com/miniconda/Miniconda3-py310_23.3.1-0-Linux-x86_64.sh"; \
    SHA256SUM="aef279d6baea7f67940f16aad17ebe5f6aac97487c7c03466ff01f4819e5a651"; \
    wget "${MINICONDA_URL}" -O miniconda.sh -q && \
    echo "${SHA256SUM} miniconda.sh" > shasum && \
    sha256sum --check --status shasum && \
    mkdir -p /opt && \
    sh miniconda.sh -b -p /opt/saturncloud && \
    rm miniconda.sh shasum && \
    /opt/saturncloud/bin/conda install -c conda-forge conda-forge::libarchive=3.6.2=h039dbb9_1 conda-forge::mamba --yes && \
    ln -s /opt/saturncloud/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    find /opt/saturncloud/ -follow -type f -name '*.a' -delete && \
    find /opt/saturncloud/ -follow -type f -name '*.js.map' -delete && \
    /opt/saturncloud/bin/conda clean -afy && \
    chown -R 1000:1000 /opt/saturncloud
