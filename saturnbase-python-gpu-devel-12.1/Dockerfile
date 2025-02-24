FROM nvidia/cuda:12.1.0-cudnn8-devel-ubuntu22.04
ENV CONDA_OVERRIDE_CUDA=12.1
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
WORKDIR ${HOME}
ENTRYPOINT []

RUN sudo ln -s /opt/saturncloud /opt/conda && \
    sudo ln -s /opt/saturncloud /srv/conda
