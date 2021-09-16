FROM rocker/r-ver:4.1.0

# hadolint ignore=DL3008,DL3009
RUN apt-get update --fix-missing && apt-get install -y --no-install-recommends \
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
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN apt-get update --fix-missing && apt-get install -y --no-install-recommends curl
# Install Python --------------------------------------------------------------#

ARG PYTHON_VERSION=3.9.5
RUN curl -O https://repo.anaconda.com/miniconda/Miniconda3-4.7.12.1-Linux-x86_64.sh && \
    bash Miniconda3-4.7.12.1-Linux-x86_64.sh -bp /opt/python/${PYTHON_VERSION} && \
    /opt/python/${PYTHON_VERSION}/bin/conda install -y python==${PYTHON_VERSION} && \
    /opt/python/${PYTHON_VERSION}/bin/pip install \
        ipykernel \
        virtualenv \
        && \
    rm -rf Miniconda3-*-Linux-x86_64.sh && \
    /opt/python/${PYTHON_VERSION}/bin/python -m ipykernel install --name py${PYTHON_VERSION} --display-name "Python ${PYTHON_VERSION}"

# Install RStudio Server Pro --------------------------------------------------#
ARG RSW_VERSION=1.4.1717-3
ARG RSW_DOWNLOAD_URL=https://download2.rstudio.org/server/bionic/amd64
ARG RSW_NAME=rstudio-workbench
RUN apt-get update --fix-missing \
    && RSW_VERSION_URL=`echo -n "${RSW_VERSION}" | sed 's/+/%2B/g'` \
    && curl -o rstudio-workbench.deb ${RSW_DOWNLOAD_URL}/${RSW_NAME}-${RSW_VERSION_URL}-amd64.deb \
    && gdebi --non-interactive rstudio-workbench.deb \
    && rm rstudio-workbench.deb \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /var/lib/rstudio-server/r-versions

# Runtime settings ------------------------------------------------------------#
ARG TINI_VERSION=0.18.0
RUN curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini && \
    chmod +x /usr/local/bin/tini

RUN curl -L -o /usr/local/bin/wait-for-it.sh https://raw.githubusercontent.com/vishnubob/wait-for-it/master/wait-for-it.sh && \
    chmod +x /usr/local/bin/wait-for-it.sh

# Copy startup
COPY startup.sh /usr/local/bin/startup.sh
RUN chmod +x /usr/local/bin/startup.sh

COPY rserver.conf /etc/rstudio/rserver.conf
COPY rstudio-prefs.json /etc/rstudio/rstudio-prefs.json

# IP address of the licensing server
ENV RSW_LICENSE_SERVER ""

# the user info for the RStudio user
ENV RSW_USER saturn
ENV RSW_USER_PASSWD saturn
ENV RSW_USER_UID 10000

# the port the container listens to. Expose whatever port you set here.
ENV RSW_PORT = 8787 

ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/startup.sh"]