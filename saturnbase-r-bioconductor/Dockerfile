FROM bioconductor/bioconductor_docker:RELEASE_3_15
RUN usermod -l jovyan rstudio
RUN groupmod -n jovyan rstudio
RUN usermod -d /home/jovyan jovyan

EXPOSE 8888

LABEL org.opencontainers.image.licenses="GPL-2.0-or-later" \
      org.opencontainers.image.source="https://github.com/saturncloud/images" \
      org.opencontainers.image.vendor="Saturn Cloud" \
      org.opencontainers.image.authors="Jacqueline Nolis" \
      org.opencontainers.image.description="Additions to the Bioconductor Docker image to run on Saturn Cloud"

# SETUP SATURN (and install linux libraries for R & Rstudio)

ENV APP_BASE=/srv
ENV CONDA_DIR=/srv/conda
ENV CONDA_BIN=${CONDA_DIR}/bin
ENV DEBIAN_FRONTEND=noninteractive
ENV LC_ALL en_US.UTF-8
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US.UTF-8
ENV SHELL /bin/bash
ENV NB_USER=jovyan
ENV NB_UID=1000
ENV USER=${NB_USER}
ENV HOME=/home/${NB_USER}
ENV CRAN=https://packagemanager.rstudio.com/cran/__linux__/focal/latest
ENV CRAN_FIXED=https://packagemanager.rstudio.com/cran/__linux__/focal/2022-04-01

ARG TINI_VERSION=0.18.0

RUN \
    # Make all library folders readable then let R known, then set up reticulate package
    mkdir -p "/usr/local/lib/R/site-library" \
    && chown 1000:1000 -R /usr/local/lib/R \
    && chmod 777 -R /usr/local/lib/R \
    && su -c "echo 'RETICULATE_PYTHON=/opt/saturncloud/envs/saturn/bin/python' >> /usr/local/lib/R/etc/Renviron.site" \
    && su -c "echo 'RSTUDIO_PANDOC=/usr/lib/rstudio-server/bin/pandoc' >> /usr/local/lib/R/etc/Renviron.site" \
    && apt-get -qq --allow-releaseinfo-change update \
    # Install packages
    && apt-get update --fix-missing \
    && apt-get upgrade -y \
    && apt-get install --yes --no-install-recommends \
        awscli \
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
        libssl1.1 \
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
        libglpk-dev \
     > /dev/null

RUN \
    sudo apt purge \
    && sudo apt clean \
    && sudo rm -rf /var/lib/apt/lists/* \
    && sudo rm -rf /var/lib/rstudio-server/r-versions \
    # Runtime settings ------------------------------------------------------------#
    && sudo curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini \
    && sudo chmod +x /usr/local/bin/tini \
    # add snowflake ODBC driver
    && sudo apt-get install -f \
    && wget https://sfc-repo.snowflakecomputing.com/odbc/linux/2.24.0/snowflake-odbc-2.24.0.x86_64.deb -O snowflake-odbc.deb \
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
    # Give user sudo access
    && echo "$NB_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook \
    && mkdir -p ${APP_BASE} \
    && mkdir -p /opt/saturncloud \
    && ln -s /opt/saturncloud /srv/conda \
    # Give user ownership of conda and app
    && chown 1000:1000 -R /opt/saturncloud \
    && chown -R $NB_USER:$NB_USER ${APP_BASE} \
    # Set the default repo to the latest rstudio package manager
    && grep -v "options(repos = c(CRAN" "${R_HOME}/etc/Rprofile.site" > tmp.txt && mv tmp.txt "${R_HOME}/etc/Rprofile.site" \
    && echo "options(repos = c(CRAN = '${CRAN}'), download.file.method = 'libcurl')" >>"${R_HOME}/etc/Rprofile.site"


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

COPY install-miniconda.bash /tmp/install-miniconda.bash
COPY setup-conda.bash /tmp/setup-conda.bash
COPY environment.yml /tmp/environment.yml

# Install miniconda
RUN bash /tmp/install-miniconda.bash && \
    bash /tmp/setup-conda.bash && \
    echo '' > ${CONDA_DIR}/conda-meta/history && \
    ${CONDA_BIN}/conda config --system --add channels conda-forge && \
    ${CONDA_BIN}/conda config --system --set auto_update_conda false && \
    ${CONDA_BIN}/conda config --system --set show_channel_urls true
ENV NPM_DIR ${APP_BASE}/npm
ENV NB_PYTHON_PREFIX ${CONDA_DIR}/envs/saturn
ENV PATH ${NB_PYTHON_PREFIX}/bin:${CONDA_BIN}:${NPM_DIR}/bin:${HOME}/.local/bin:${PATH}
WORKDIR ${HOME}

# Configure rstudio
COPY --chown=root:root rstudio-start.sh /usr/local/bin/rstudio-start.sh
RUN sudo chmod +x /usr/local/bin/rstudio-start.sh

COPY --chown=root:root database.conf /etc/rstudio/database.conf
COPY --chown=root:root rserver.conf /etc/rstudio/rserver.conf
COPY --chown=root:root rstudio-prefs.json /etc/rstudio/rstudio-prefs.json

ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/rstudio-start.sh"]
