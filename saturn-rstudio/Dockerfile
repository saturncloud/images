ARG SATURN_R_IMAGE
FROM ${SATURN_R_IMAGE}

ARG RSW_VERSION=1.4.1717-3
ARG RSW_DOWNLOAD_URL=https://download2.rstudio.org/server/bionic/amd64
ARG RSW_NAME=rstudio-workbench

ARG TINI_VERSION=0.18.0

# hadolint ignore=DL3008,DL3009
RUN sudo apt-get update --fix-missing \
    && sudo apt-get install -y --no-install-recommends \
        bzip2 \
        ca-certificates \
        gdebi-core \
        git \
        libcap2 \
        libglib2.0-0 \
        libpq5 \
        libsm6 \
        libssl1.1 \
        openssl \
        libssl-dev \
        libuser \
        libuser1-dev \
        libxext6 \
        libxrender1 \
        openssh-client \
        rrdtool \
        sudo \
        wget \
        curl \
    # Install RStudio Server Pro --------------------------------------------------#
    && RSW_VERSION_URL=`echo -n "${RSW_VERSION}" | sed 's/+/%2B/g'` \
    && curl -o rstudio-workbench.deb ${RSW_DOWNLOAD_URL}/${RSW_NAME}-${RSW_VERSION_URL}-amd64.deb \
    && sudo gdebi --non-interactive rstudio-workbench.deb \
    && rm rstudio-workbench.deb \
    && sudo apt clean \
    && sudo rm -rf /var/lib/apt/lists/* \
    && sudo rm -rf /var/lib/rstudio-server/r-versions \
    # Runtime settings ------------------------------------------------------------#
    && sudo curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini \
    && sudo chmod +x /usr/local/bin/tini

# Copy startup
COPY --chown=root:root startup.sh /usr/local/bin/startup.sh
RUN sudo chmod +x /usr/local/bin/startup.sh

COPY --chown=root:root rserver.conf /etc/rstudio/rserver.conf
COPY --chown=root:root rstudio-prefs.json /etc/rstudio/rstudio-prefs.json

# IP address of the licensing server
ENV RSW_LICENSE_SERVER ""

# the port the container listens to. Expose whatever port you set here.
ENV RSW_PORT 8787 

# ENTRYPOINT ["tini", "--"]
CMD ["sudo", "-E", "/usr/local/bin/startup.sh"]
