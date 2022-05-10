FROM ubuntu:focal
EXPOSE 8888

# SETUP SATURN

ENV APP_BASE=/srv
ENV CONDA_DIR=/srv/conda
ENV CONDA_BIN=${CONDA_DIR}/bin
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get -qq --allow-releaseinfo-change update && \
    apt-get -qq upgrade -y && \
    apt-get -qq install --yes --no-install-recommends \
        awscli \
        build-essential \
        bzip2 \
        ca-certificates \
        curl \
        htop \
        gettext-base \
        git \
        gnupg \
        locales \
        openssh-client \
        openssh-server \
        procps \
        rsync \
        sudo \
        screen \
        wget \
        devscripts \
        libxml2 \
        libxml2-dev \
        libcurl4-openssl-dev \
        libssl-dev \
        libzmq3-dev \
        qpdf \
        gdebi-core \
        libcap2 \
        libglib2.0-0 \
        libpq5 \
        libsm6 \
        libcurl4-openssl-dev \
        libssl1.1 \
        openssl \
        libuser \
        libuser1-dev \
        openssh-client \
        rrdtool \
        unixodbc \
        unixodbc-dev \
        pandoc \
        libglpk-dev \
     > /dev/null && \
    # add snowflake ODBC driver
    sudo apt-get install -f && \
    wget https://sfc-repo.snowflakecomputing.com/odbc/linux/2.24.0/snowflake-odbc-2.24.0.x86_64.deb -O snowflake-odbc.deb && \
    sudo dpkg -i snowflake-odbc.deb && \
    sudo wget https://saturn-public-data.s3.us-east-2.amazonaws.com/r-odbc/odbc.ini -O /usr/lib/snowflake/odbc/lib/odbc.ini && \
    sudo wget https://saturn-public-data.s3.us-east-2.amazonaws.com/r-odbc/odbcinst.ini -O /usr/lib/snowflake/odbc/lib/odbcinst.ini && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /run/sshd && \
    chmod 755 /run/sshd && \
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen

ENV PATH /opt/conda/bin:$PATH

# Based on https://github.com/ContinuumIO/docker-images/blob/4d7798c0ea2463d9c4057d8eaee876102eecbf86/miniconda3/debian/Dockerfile
ARG CONDA_VERSION=py39_4.10.3
RUN set -x && \
    MINICONDA_URL="https://repo.anaconda.com/miniconda/Miniconda3-${CONDA_VERSION}-Linux-x86_64.sh"; \
    SHA256SUM="1ea2f885b4dbc3098662845560bc64271eb17085387a70c2ba3f29fff6f8d52f"; \
    wget "${MINICONDA_URL}" -O miniconda.sh -q && \
    echo "${SHA256SUM} miniconda.sh" > shasum && \
    if [ "${CONDA_VERSION}" != "latest" ]; then sha256sum --check --status shasum; fi && \
    mkdir -p /opt && \
    sh miniconda.sh -b -p /opt/conda && \
    rm miniconda.sh shasum && \
    ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    echo ". /opt/conda/etc/profile.d/conda.sh" >> ~/.bashrc && \
    echo "conda activate base" >> ~/.bashrc && \
    find /opt/conda/ -follow -type f -name '*.a' -delete && \
    find /opt/conda/ -follow -type f -name '*.js.map' -delete && \
    /opt/conda/bin/conda clean -afy && \
    ln -s /opt/conda /srv/conda && \
    chown -R 1000:1000 /opt/conda && \
    chown -R 1000:1000 /srv

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

## Install R

RUN sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-key 'E298A3A825C0D65DFD57CBB651716619E084DAB9' && \
    sudo su -c \
    "echo 'deb https://cloud.r-project.org/bin/linux/ubuntu focal-cran40/' >>/etc/apt/sources.list" && \
    sudo apt update && \
    sudo apt-get install -y -t focal-cran40 \
        r-base=4.2.0-1~bullseyecran.0 && \
    sudo apt-mark hold r-base && \
    sudo apt-get install -y -t focal-cran40 \
        r-base-core \
        r-base-dev 

# Make all library folders readable then let R known, the set up reticulate and pandoc packages
RUN sudo mkdir -p /usr/lib/R/site-library \
    && sudo mkdir -p /usr/lib/R/library \
    && sudo chown 1000:1000 -R /usr/lib/R \
    && sudo chmod 777 -R /usr/lib/R \
    && sudo su -c "echo 'R_LIBS=/usr/lib/R/site-library:/usr/lib/R/library' >> /usr/lib/R/etc/Renviron.site" \
    && sudo su -c "echo 'RETICULATE_PYTHON=/opt/conda/envs/saturn/bin/python' >> /usr/lib/R/etc/Renviron.site" \
    && sudo su -c "echo 'options(repos = c(CRAN = \"https://packagemanager.rstudio.com/cran/__linux__/focal/latest\"), download.file.method = \"libcurl\")' >> /usr/lib/R/etc/Rprofile.site" \
    && sudo su -c "echo 'options(HTTPUserAgent = sprintf(\"R/%s R (%s)\", getRversion(), paste(getRversion(), R.version\$platform, R.version\$arch, R.version\$os)))' >> /usr/lib/R/etc/Rprofile.site"

## Install RStudio

ARG RSTUDIO_VERSION=2022.02.1-461

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
        'tinytex'), \
        lib = '/usr/lib/R/site-library')" \
    && sudo rm -rf /tmp/*

COPY --chown=root:root rstudio-start.sh /usr/local/bin/rstudio-start.sh
RUN sudo chmod +x /usr/local/bin/rstudio-start.sh

COPY --chown=root:root database.conf /etc/rstudio/database.conf
COPY --chown=root:root rserver.conf /etc/rstudio/rserver.conf
COPY --chown=root:root rstudio-prefs.json /etc/rstudio/rstudio-prefs.json

ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/rstudio-start.sh"]
