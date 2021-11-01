ARG SATURN_R_IMAGE
FROM ${SATURN_R_IMAGE}

ARG RSTUDIO_VERSION=2021.09.0-351

ARG TINI_VERSION=0.18.0

# hadolint ignore=DL3008,DL3009
RUN sudo apt-get update --fix-missing \
    && sudo update-locale LANG=en_US.utf8 \
    && sudo apt-get install -y --no-install-recommends \
        # bzip2 \
        # ca-certificates \
        gdebi-core \
        # git \
        libcap2 \
        libglib2.0-0 \
        libpq5 \
        libsm6 \
        libcurl4-openssl-dev \
        libssl1.1 \
        openssl \
        libssl-dev \
        libuser \
        libuser1-dev \
        # libxext6 \
        # libxrender1 \
        openssh-client \
        rrdtool \
        # sudo \
        # wget \
        # curl \
    # Install RStudio Server --------------------------------------------------#
    && curl -o rstudio.deb https://download2.rstudio.org/server/bionic/amd64/rstudio-server-${RSTUDIO_VERSION}-amd64.deb \
    && sudo gdebi --non-interactive rstudio.deb \
    && rm rstudio.deb \
    && sudo apt clean \
    && sudo rm -rf /var/lib/apt/lists/* \
    && sudo rm -rf /var/lib/rstudio-server/r-versions \
    # Runtime settings ------------------------------------------------------------#
    && sudo curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini \
    && sudo chmod +x /usr/local/bin/tini


COPY --chown=root:root rstudio-start.sh /usr/local/bin/rstudio-start.sh
RUN sudo chmod +x /usr/local/bin/rstudio-start.sh

COPY --chown=root:root database.conf /etc/rstudio/database.conf
COPY --chown=root:root rserver.conf /etc/rstudio/rserver.conf
# COPY --chown=root:root rsession-profile /etc/rstudio/rsession-profile
COPY --chown=root:root rstudio-prefs.json /etc/rstudio/rstudio-prefs.json

ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/rstudio-start.sh"]
