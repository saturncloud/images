FROM nvidia/cuda:13.3.0-cudnn-runtime-ubuntu22.04
ENV CONDA_OVERRIDE_CUDA=13.3
EXPOSE 8888


ENV CONDA_DIR=/opt/saturncloud
ENV CONDA_BIN=${CONDA_DIR}/bin
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get -qq --allow-releaseinfo-change update && \
    apt-get -qq upgrade -y && \
    apt-get -qq install --yes --no-install-recommends \
    awscli \
    dumb-init \
    build-essential \
    bzip2 \
    ca-certificates \
    curl \
    file \
    gettext-base \
    git \
    gnupg \
    htop \
    libnuma-dev \
    locales \
    openssh-server \
    rsync \
    screen \
    ssh \
    sudo \
    unzip \
    wget \
    zip \
    > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /run/sshd && \
    chmod 755 /run/sshd && \
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen
ENV LC_ALL en_US.UTF-8
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US.UTF-8
ENV SHELL /bin/bash
ENV NB_USER=jovyan
ENV NB_UID=1000

COPY install-miniconda.bash /tmp/install-miniconda.bash
RUN bash /tmp/install-miniconda.bash

ENV USER=${NB_USER}
ENV HOME=/home/${NB_USER}

RUN adduser --disabled-password \
    --gecos "Default user" \
    --uid ${NB_UID} \
    ${NB_USER} && \
    echo "$NB_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook && \
    chown 1000:1000 -R /opt/saturncloud

COPY profile /etc/profile

USER ${NB_USER}


COPY install-jupyter.bash /tmp/install-jupyter.bash
COPY environment.yml /tmp/environment.yml
RUN bash /tmp/install-jupyter.bash && \
    echo '' > ${CONDA_DIR}/conda-meta/history && \
    ${CONDA_BIN}/conda config --system --add channels conda-forge && \
    ${CONDA_BIN}/conda config --system --set auto_update_conda false && \
    ${CONDA_BIN}/conda config --system --set show_channel_urls true
ENV NB_PYTHON_PREFIX ${CONDA_DIR}/envs/saturn
ENV PATH ${NB_PYTHON_PREFIX}/bin:${CONDA_BIN}:${HOME}/.local/bin:${PATH}
# Prefer the conda env's libstdc++ over the CUDA base's system one. The
# nvidia/cuda:13.3-ubuntu22.04 base ships an older system libstdc++ (max
# CXXABI_1.3.13); newer compiled wheels (e.g. scipy's _highspy, pulled in by
# vLLM's transformers import chain) need CXXABI_1.3.15, which the conda env's
# libstdcxx-ng DOES provide. Without this, `vllm serve` crashes at import with
# `libstdc++.so.6: version CXXABI_1.3.15 not found`. (On the CUDA-12 bases this
# was masked because torch/libcudart failed earlier; on CUDA 13 the import gets
# far enough to hit it.)
ENV LD_LIBRARY_PATH ${NB_PYTHON_PREFIX}/lib:${CONDA_DIR}/lib${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}
WORKDIR ${HOME}
ENTRYPOINT []

RUN sudo ln -s /opt/saturncloud /opt/conda && \
    sudo ln -s /opt/saturncloud /srv/conda