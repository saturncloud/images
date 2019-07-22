FROM nvidia/cuda:10.0-devel-ubuntu18.04
EXPOSE 8888
ENV APP_BASE=/srv
ENV CONDA_DIR=/srv/conda
ENV PATH=${PATH}:/srv/conda/bin
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends \
        locales \
        wget \
        curl \
     > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*
RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
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
    ${NB_USER}
RUN wget --quiet -O - https://deb.nodesource.com/gpgkey/nodesource.gpg.key |  apt-key add - && \
    DISTRO="bionic" && \
    echo "deb https://deb.nodesource.com/node_10.x $DISTRO main" >> /etc/apt/sources.list.d/nodesource.list && \
    echo "deb-src https://deb.nodesource.com/node_10.x $DISTRO main" >> /etc/apt/sources.list.d/nodesource.list

RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends \
        less \
        nodejs \
        unzip \
        bzip2 \
        ca-certificates \
        sudo \
        fonts-liberation \
        git \
        emacs-nox \
        vim \
        ssh \
        build-essential \
        awscli \
        gcc-5 \
        g++-5 \
        zlib1g-dev \
     > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*
RUN echo "$NB_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook
RUN mkdir -p ${APP_BASE} && \
    chown -R $NB_USER:$NB_USER ${APP_BASE}
COPY install-jupyter.bash /tmp/install-jupyter.bash
COPY jupyter.yml /tmp/jupyter.yml
RUN bash /tmp/install-jupyter.bash
RUN ${CONDA_DIR}/bin/jupyter serverextension enable --py nbserverproxy
RUN ${CONDA_DIR}/bin/jupyter labextension install jupyterlab_bokeh
RUN ${CONDA_DIR}/bin/conda create -n saturn
RUN ${CONDA_DIR}/bin/conda clean --all
WORKDIR ${HOME}

ENV NPM_DIR ${APP_BASE}/npm
ENV NB_PYTHON_PREFIX ${CONDA_DIR}/envs/saturn
ENV PATH ${NB_PYTHON_PREFIX}/bin:${CONDA_DIR}/bin:${NPM_DIR}/bin:${HOME}/.local/bin:${PATH}
