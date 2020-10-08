FROM continuumio/miniconda3:4.8.2
EXPOSE 8888

ARG JUPYTER_SATURN_VERSION=1.2.7

ENV JUPYTER_SATURN_VERSION=${JUPYTER_SATURN_VERSION}

RUN ln -s /opt/conda /srv/conda
RUN chown 1000:1000 -R /opt/conda
ENV APP_BASE=/srv
ENV CONDA_DIR=/srv/conda
ENV PATH=${PATH}:/srv/conda/bin
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends \
        gettext-base \
        gnupg \
        locales \
        wget \
        curl \
        bzip2 \
        ca-certificates \
        git \
        awscli \
        rsync \
        sudo \
        openssh-server \
        htop \
        screen \
     > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /run/sshd && \
    chmod 755 /run/sshd
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

RUN echo "$NB_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook
RUN mkdir -p ${APP_BASE} && \
    chown -R $NB_USER:$NB_USER ${APP_BASE}
COPY profile /etc/profile

USER ${NB_USER}

COPY install-jupyter.bash /tmp/install-jupyter.bash
COPY environment.yml /tmp/environment-temp.yml
RUN envsubst < /tmp/environment-temp.yml > /tmp/environment.yml
RUN bash /tmp/install-jupyter.bash
RUN echo '' > ${CONDA_DIR}/conda-meta/history
RUN conda config --system --add channels conda-forge && \
    conda config --system --add channels https://conda.saturncloud.io/pkgs && \
    conda config --system --set auto_update_conda false && \
    conda config --system --set show_channel_urls true
ENV NPM_DIR ${APP_BASE}/npm
ENV NB_PYTHON_PREFIX ${CONDA_DIR}/envs/saturn
ENV PATH ${NB_PYTHON_PREFIX}/bin:${CONDA_DIR}/bin:${NPM_DIR}/bin:${HOME}/.local/bin:${PATH}
WORKDIR ${HOME}
