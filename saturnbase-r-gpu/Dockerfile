FROM rocker/cuda:4.3
ENV CONDA_OVERRIDE_CUDA=11.8
EXPOSE 8888

LABEL org.opencontainers.image.licenses="GPL-2.0-or-later" \
      org.opencontainers.image.source="https://github.com/saturncloud/images" \
      org.opencontainers.image.vendor="Saturn Cloud" \
      org.opencontainers.image.authors="Jacqueline Nolis" \
      org.opencontainers.image.description="Additions to rocker/cuda to support running on Saturn Cloud"

# SETUP SATURN (and install linux libraries for R & Rstudio)

ENV CONDA_DIR=/opt/saturncloud
ENV CONDA_BIN=${CONDA_DIR}/bin
ENV DEBIAN_FRONTEND=noninteractive
ENV LC_ALL en_US.UTF-8
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US.UTF-8
ENV SHELL /bin/bash
ENV NB_USER=jovyan
ENV NB_UID=1000
ENV CRAN=https://packagemanager.rstudio.com/cran/__linux__/jammy/latest
ENV CRAN_FIXED=https://packagemanager.rstudio.com/cran/__linux__/jammy/latest

ARG RSTUDIO_VERSION=2023.06.0-421

ARG TINI_VERSION=0.18.0

RUN apt-key del 7fa2af80 \
    && rm -f /etc/apt/sources.list.d/nvidia-ml.list /etc/apt/sources.list.d/cuda.list \
    && apt-get -qq --allow-releaseinfo-change update \
    # Make all library folders readable then let R known, then set up reticulate package
    && mkdir -p "/usr/local/lib/R/site-library" \
    && chown 1000:1000 -R /usr/local/lib/R \
    && chmod 777 -R /usr/local/lib/R \
    && apt-get -qq --allow-releaseinfo-change update \
    # Install packages
    && apt-get update --fix-missing \
    && apt-get upgrade -y \
    && apt-get install --yes --no-install-recommends \
        awscli \
        dumb-init \
        build-essential \
        bzip2 \
        ca-certificates \
        curl \
        devscripts \
        file \
        gdebi-core \
        gettext-base \
        git \
        gnupg \
        htop \
        libcap2 \
        libcurl4-openssl-dev \
        libglib2.0-0 \
        libpq5 \
        libsm6 \
        libssl-dev \
        libuser \
        libuser1-dev \
        libxml2 \
        libxml2-dev \
        libzmq3-dev \
        locales \
        openssh-client \
        openssh-server \
        openssl \
        pandoc \
        psmisc \
        qpdf \
        rrdtool \
        rsync \
        screen \
        ssh \
        sudo \
        wget \
        unixodbc \
        unixodbc-dev \
        odbcinst \
        libglpk-dev \
     > /dev/null \
    && apt-get install --yes --no-install-recommends libssl-dev  > /dev/null \
    # Install rstudio-server
    && curl -o rstudio.deb https://download2.rstudio.org/server/jammy/amd64/rstudio-server-2024.04.2-764-amd64.deb \
    && sudo gdebi --non-interactive rstudio.deb \
    && rm rstudio.deb \
    && sudo apt purge \
    && sudo apt clean \
    && sudo rm -rf /var/lib/apt/lists/* \
    && sudo rm -rf /var/lib/rstudio-server/r-versions \
    # Runtime settings ------------------------------------------------------------#
    && sudo curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini \
    && sudo chmod +x /usr/local/bin/tini \
    # add snowflake ODBC driver
    && sudo apt-get install -f \
    && wget https://sfc-repo.snowflakecomputing.com/odbc/linux/2.25.12/snowflake-odbc-2.25.12.x86_64.deb -O snowflake-odbc.deb \
    && SHA256SUM="3349246214bb990414bf9fb2b440eb63b9c0197765e714054b651e72406a1df0" \
    && echo "${SHA256SUM} snowflake-odbc.deb" > shasum \
    && sha256sum --check --status shasum \
    && sudo dpkg -i snowflake-odbc.deb \
    && sudo wget https://saturn-public-data.s3.us-east-2.amazonaws.com/r-odbc/odbc.ini -O /usr/lib/snowflake/odbc/lib/odbc.ini \
    && sudo wget https://saturn-public-data.s3.us-east-2.amazonaws.com/r-odbc/odbcinst.ini -O /usr/lib/snowflake/odbc/lib/odbcinst.ini \
    # Cleanup
    && sudo rm -rf /tmp/* \
    && apt-get -qq purge \
    && apt-get -qq clean \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /run/sshd \
    && chmod 755 /run/sshd \
    # Generate locales
    && echo "en_US.UTF-8 UTF-8" > /etc/locale.gen \
    && locale-gen \
    # Set the default repo to the latest rstudio package manager
    && grep -v "options(repos = c(CRAN" "${R_HOME}/etc/Rprofile.site" > tmp.txt && mv tmp.txt "${R_HOME}/etc/Rprofile.site" \
    && echo "options(repos = c(CRAN = '${CRAN}'), download.file.method = 'libcurl')" >>"${R_HOME}/etc/Rprofile.site"

COPY install-miniconda.bash /tmp/install-miniconda.bash
ARG CONDA_VERSION=py39_4.11.0
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

# Add a few R packages that are useful for RMarkdown
RUN Rscript -e "install.packages(c( \
        'jquerylib', \
        'markdown', \
        'rmarkdown', \
        'tinytex' \
        ), Ncpus = max(c(1, parallel::detectCores() - 1)), \
        dependencies = c('LinkingTo', 'Depends', 'Imports'), \
        lib = '/usr/local/lib/R/site-library', \
        repos = '${CRAN_FIXED}'\
        )"

COPY setup-conda.bash /tmp/setup-conda.bash

# Install miniconda
RUN bash /tmp/setup-conda.bash && \
    echo '' > ${CONDA_DIR}/conda-meta/history && \
    ${CONDA_BIN}/conda config --system --add channels conda-forge && \
    ${CONDA_BIN}/conda config --system --set auto_update_conda false && \
    ${CONDA_BIN}/conda config --system --set show_channel_urls true
ENV NB_PYTHON_PREFIX ${CONDA_DIR}/envs/saturn
ENV PATH ${NB_PYTHON_PREFIX}/bin:${CONDA_BIN}:${HOME}/.local/bin:${PATH}
ENV RETICULATE_PYTHON /opt/saturncloud/envs/saturn/bin/python
WORKDIR ${HOME}

# Configure rstudio
COPY --chown=root:root rstudio-start.sh /usr/local/bin/rstudio-start.sh
RUN sudo chmod +x /usr/local/bin/rstudio-start.sh

COPY --chown=root:root database.conf /etc/rstudio/database.conf
COPY --chown=root:root rserver.conf /etc/rstudio/rserver.conf
COPY --chown=root:root rstudio-prefs.json /etc/rstudio/rstudio-prefs.json

ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/rstudio-start.sh"]
ENTRYPOINT []
