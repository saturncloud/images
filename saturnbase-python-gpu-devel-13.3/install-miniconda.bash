#!/bin/bash
cd /tmp

# One installer per CPU architecture: the CUDA base publishes linux/amd64 and
# linux/arm64 (Grace GPU nodes), and the installer must match the build platform.
case "$(uname -m)" in
  x86_64)
    MINICONDA_URL="https://repo.anaconda.com/miniconda/Miniconda3-py311_24.4.0-0-Linux-x86_64.sh"
    SHA256SUM="7cb030a12d1da35e1c548344a895b108e0d2fbdc4f6b67d5180b2ac8539cc473"
    ;;
  aarch64)
    MINICONDA_URL="https://repo.anaconda.com/miniconda/Miniconda3-py311_24.4.0-0-Linux-aarch64.sh"
    SHA256SUM="d3ed9ff565c7341a4e4c957443ccb4f12f3dd5a087549099e7f199c36469e55c"
    ;;
  *)
    echo "unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

set -x && \
  wget "${MINICONDA_URL}" -O miniconda.sh -q && \
  echo "${SHA256SUM} miniconda.sh" > shasum && \
  sha256sum --check --status shasum && \
  mkdir -p /opt && \
  sh miniconda.sh -b -p /opt/saturncloud && \
  rm miniconda.sh shasum && \
  /opt/saturncloud/bin/conda install -n base --override-channels -c conda-forge mamba==1.5.10 'python_abi=*=*cp*' --yes
  ln -s /opt/saturncloud/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
  find /opt/saturncloud/ -follow -type f -name '*.a' -delete && \
  find /opt/saturncloud/ -follow -type f -name '*.js.map' -delete && \
  /opt/saturncloud/bin/conda clean -afy && \
  chown -R 1000:1000 /opt/saturncloud
