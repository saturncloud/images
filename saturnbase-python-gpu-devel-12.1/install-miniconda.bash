#!/bin/bash
cd /tmp

set -x && \
  MINICONDA_URL="https://repo.anaconda.com/miniconda/Miniconda3-py311_24.4.0-0-Linux-x86_64.sh"; \
  SHA256SUM="7cb030a12d1da35e1c548344a895b108e0d2fbdc4f6b67d5180b2ac8539cc473"; \
  wget "${MINICONDA_URL}" -O miniconda.sh -q && \
  echo "${SHA256SUM} miniconda.sh" > shasum && \
  sha256sum --check --status shasum && \
  mkdir -p /opt && \
  sh miniconda.sh -b -p /opt/saturncloud && \
  rm miniconda.sh shasum && \
  /opt/saturncloud/bin/conda install -n base python=3.11.8 conda=24.9.2 libsqlite=3.45.2 -c conda-forge && \
  ln -s /opt/saturncloud/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
  find /opt/saturncloud/ -follow -type f -name '*.a' -delete && \
  find /opt/saturncloud/ -follow -type f -name '*.js.map' -delete && \
  /opt/saturncloud/bin/conda clean -afy && \
  chown -R 1000:1000 /opt/saturncloud
