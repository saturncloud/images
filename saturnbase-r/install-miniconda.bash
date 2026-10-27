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
  # CVE-2026-111943: the installer's pip 24.0 vendors a certifi with the emSign ECC Root CA - G3;
  # 26.2.2 vendors 2026.10.19 (same pin as saturnbase-python's environment.yml).
  /opt/saturncloud/bin/conda install -n base --override-channels -c conda-forge mamba==1.5.10 'python_abi=*=*cp*' 'pip>=26.2.2' --yes
  ln -s /opt/saturncloud/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
  find /opt/saturncloud/ -follow -type f -name '*.a' -delete && \
  find /opt/saturncloud/ -follow -type f -name '*.js.map' -delete && \
  /opt/saturncloud/bin/conda clean -afy && \
  chown -R 1000:1000 /opt/saturncloud