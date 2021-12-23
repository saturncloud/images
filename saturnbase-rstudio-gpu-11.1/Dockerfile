FROM rocker/cuda:cuda11.1
EXPOSE 8888

LABEL org.opencontainers.image.licenses="GPL-2.0-or-later" \
      org.opencontainers.image.source="https://github.com/saturncloud/images" \
      org.opencontainers.image.vendor="Saturn Cloud" \
      org.opencontainers.image.authors="Jacqueline Nolis" \
      org.opencontainers.image.description="Additions to rocker/cuda to support running on Saturn Cloud"

# SETUP SATURN (and install linux libraries for R & Rstudio)

ARG JUPYTER_SATURN_VERSION

ENV JUPYTER_SATURN_VERSION=${JUPYTER_SATURN_VERSION}

RUN mkdir /opt/conda && \
    ln -s /opt/conda /srv/conda
ENV APP_BASE=/srv
ENV CONDA_DIR=/srv/conda
ENV CONDA_BIN=${CONDA_DIR}/bin
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get -qq --allow-releaseinfo-change update && \
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
        devscripts \
        libxml2 \
        libxml2-dev \
        libcurl4-openssl-dev \
        libzmq3-dev \
        qpdf \
        gdebi-core \
        libcap2 \
        libglib2.0-0 \
        libpq5 \
        libsm6 \
        libssl1.1 \
        openssl \
        libssl-dev \
        libuser \
        libuser1-dev \
        openssh-client \
        rrdtool \
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
ENV USER=${NB_USER}
ENV HOME=/home/${NB_USER}

RUN adduser --disabled-password \
    --gecos "Default user" \
    --uid ${NB_UID} \
    ${NB_USER} && \
    echo "$NB_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook && \
    mkdir -p ${APP_BASE} && \
    chown 1000:1000 -R /opt/conda && \
    chown -R $NB_USER:$NB_USER ${APP_BASE}
USER ${NB_USER}

COPY install-miniconda.bash /tmp/install-miniconda.bash
RUN bash /tmp/install-miniconda.bash

COPY setup-conda.bash /tmp/setup-conda.bash
COPY environment.yml /tmp/environment.yml
RUN bash /tmp/setup-conda.bash && \
    echo '' > ${CONDA_DIR}/conda-meta/history && \
    ${CONDA_BIN}/conda config --system --add channels conda-forge && \
    ${CONDA_BIN}/conda config --system --set auto_update_conda false && \
    ${CONDA_BIN}/conda config --system --set show_channel_urls true
ENV NPM_DIR ${APP_BASE}/npm
ENV NB_PYTHON_PREFIX ${CONDA_DIR}/envs/saturn
ENV PATH ${NB_PYTHON_PREFIX}/bin:${CONDA_BIN}:${NPM_DIR}/bin:${HOME}/.local/bin:${PATH}
WORKDIR ${HOME}


# CUDA FIX
RUN sudo ln -s /usr/local/cuda-11.1/targets/x86_64-linux/lib/libcusolver.so.11 /usr/local/lib/libcusolver.so.10

# ADDITIONAL R SETUP BEYOND BASE IMAGE

# Make all library folders readable then let R known, the set up reticulate package
RUN sudo mkdir -p /usr/local/lib/R \
    && sudo mkdir -p /usr/local/lib/R/site-library \
    && sudo mkdir -p /usr/lib/R/site-library \
    && sudo mkdir -p /usr/lib/R/library \
    && sudo chown 1000:1000 -R /usr/local/lib/R \
    && sudo chmod 777 -R /usr/local/lib/R \
    && sudo chown 1000:1000 -R /usr/lib/R \
    && sudo chmod 777 -R /usr/lib/R \
    && sudo su -c "echo 'R_LIBS=/usr/local/lib/R:/usr/local/lib/R/site-library:/usr/lib/R/site-library:/usr/lib/R/library' >> /usr/local/lib/R/etc/Renviron" \
    && sudo su -c "echo 'RETICULATE_PYTHON=/opt/conda/envs/saturn/bin/python' >> /usr/local/lib/R/etc/Renviron"

## Install RStudio

ARG RSTUDIO_VERSION=2021.09.0-351

ARG TINI_VERSION=0.18.0

# hadolint ignore=DL3008,DL3009
RUN curl -o rstudio.deb https://download2.rstudio.org/server/bionic/amd64/rstudio-server-${RSTUDIO_VERSION}-amd64.deb \
    && sudo gdebi --non-interactive rstudio.deb \
    && rm rstudio.deb \
    && sudo apt purge \
    && sudo apt clean \
    && sudo rm -rf /var/lib/apt/lists/* \
    && sudo rm -rf /var/lib/rstudio-server/r-versions \
    # Runtime settings ------------------------------------------------------------#
    && sudo curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini \
    && sudo chmod +x /usr/local/bin/tini \
    # Add a few R packages that are useful for RMarkdown
    && Rscript -e "install.packages(c( \
        'jquerylib', \
        'markdown', \
        'rmarkdown', \
        'tinytex' \
        ), Ncpus = max(c(1, parallel::detectCores() - 1)), \
        dependencies = c('LinkingTo', 'Depends', 'Imports'), \
        lib = '/usr/local/lib/R/site-library' \
        )" \
    && sudo rm -rf /tmp/*

COPY --chown=root:root rstudio-start.sh /usr/local/bin/rstudio-start.sh
RUN sudo chmod +x /usr/local/bin/rstudio-start.sh

COPY --chown=root:root database.conf /etc/rstudio/database.conf
COPY --chown=root:root rserver.conf /etc/rstudio/rserver.conf
COPY --chown=root:root rstudio-prefs.json /etc/rstudio/rstudio-prefs.json

ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/rstudio-start.sh"]