FROM daskdev/dask:latest

RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends locales > /dev/null && \
    apt-get -y install wget git curl cron unzip && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*
ENV SHELL /bin/bash

RUN conda update dask=2.8.1
RUN conda update distributed

RUN conda install -y -c conda-forge dask-kubernetes
RUN conda install -y -c https://conda.saturncloud.io/pkgs sutils=0.5.1
RUN conda install -y -c conda-forge kubernetes=1.13.5
RUN conda install -y -c conda-forge python-kubernetes=9.0

RUN mkdir /root/saturn
RUN mkdir /root/saturn/pdc
RUN mkdir /root/saturn/pdc/kubet
RUN mkdir /root/saturn/pdc/dask

COPY saturn/pdc/kubet /root/saturn/pdc/kubet
COPY saturn/pdc/dask /root/saturn/pdc/dask

ENV PYTHONPATH=/root/saturn

