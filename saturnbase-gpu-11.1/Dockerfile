FROM nvidia/cuda:11.1.1-runtime-ubuntu18.04
EXPOSE 8888

ARG JUPYTER_SATURN_VERSION

ENV JUPYTER_SATURN_VERSION=${JUPYTER_SATURN_VERSION}

RUN mkdir /opt/saturncloud && \
    ln -s /opt/saturncloud /srv/conda
ENV APP_BASE=/srv
ENV CONDA_DIR=/srv/conda
ENV CONDA_BIN=${CONDA_DIR}/bin
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-key del 7fa2af80 && \
    rm /etc/apt/sources.list.d/nvidia-ml.list /etc/apt/sources.list.d/cuda.list && \
    apt-get -qq --allow-releaseinfo-change update && \
    apt-get -qq upgrade -y && \
    apt-get -qq install --yes --no-install-recommends \
        gettext-base \
        gnupg \
        locales \
        wget \
        curl \
        bzip2 \
        ca-certificates \
        ssh \
        file \
        git \
        awscli \
        rsync \
        sudo \
        openssh-server \
        htop \
        screen \
        build-essential \
        libnuma-dev \
     > /dev/null && \
    wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64/cuda-keyring_1.0-1_all.deb && \
    dpkg -i cuda-keyring_1.0-1_all.deb && \
    rm cuda-keyring_1.0-1_all.deb && \
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
ENV USER=${NB_USER}
ENV HOME=/home/${NB_USER}

RUN adduser --disabled-password \
    --gecos "Default user" \
    --uid ${NB_UID} \
    ${NB_USER} && \
    echo "$NB_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook && \
    mkdir -p ${APP_BASE} && \
    chown 1000:1000 -R /opt/saturncloud && \
    chown -R $NB_USER:$NB_USER ${APP_BASE}
USER ${NB_USER}

COPY install-miniconda.bash /tmp/install-miniconda.bash
RUN bash /tmp/install-miniconda.bash

COPY install-jupyter.bash /tmp/install-jupyter.bash
COPY environment.yml /tmp/environment.yml
RUN  bash /tmp/install-jupyter.bash && \
    echo '' > ${CONDA_DIR}/conda-meta/history && \
    ${CONDA_BIN}/conda config --system --add channels conda-forge && \
    ${CONDA_BIN}/conda config --system --set auto_update_conda false && \
    ${CONDA_BIN}/conda config --system --set show_channel_urls true
ENV NPM_DIR ${APP_BASE}/npm
ENV NB_PYTHON_PREFIX ${CONDA_DIR}/envs/saturn
ENV PATH ${NB_PYTHON_PREFIX}/bin:${CONDA_BIN}:${NPM_DIR}/bin:${HOME}/.local/bin:${PATH}
WORKDIR ${HOME}
