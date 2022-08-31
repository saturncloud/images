FROM ubuntu:focal
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

ENV PATH /opt/saturncloud/bin:$PATH

CMD [ "/bin/bash" ]

# Based on https://github.com/ContinuumIO/docker-images/blob/4d7798c0ea2463d9c4057d8eaee876102eecbf86/miniconda3/debian/Dockerfile
ARG CONDA_VERSION=py39_4.11.0
RUN set -x && \
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


ENV LC_ALL=en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US.UTF-8
ENV SHELL=/bin/bash
ENV NB_USER=jovyan
ENV NB_UID=1000
ENV USER=${NB_USER}
ENV HOME=/home/${NB_USER}

RUN adduser --disabled-password \
    --gecos "Default user" \
    --uid ${NB_UID} \
    ${NB_USER} && \
    echo "$NB_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook
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
ENV PATH ${NB_PYTHON_PREFIX}/bin:${CONDA_BIN}:${NPM_DIR}/bin:${HOME}/.local/bin:${PATH}
WORKDIR ${HOME}
