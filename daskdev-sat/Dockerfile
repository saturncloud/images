FROM daskdev/dask:2.8.1

RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends locales > /dev/null && \
    apt-get -y install wget git curl cron unzip && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*
ENV SHELL /bin/bash

RUN conda install -y dask=2.8.1
RUN conda install -y distributed

RUN conda install -y -c conda-forge dask-kubernetes
RUN conda install -y -c https://conda.saturncloud.io/pkgs sutils=0.5.1
RUN conda install -y -c conda-forge kubernetes=1.13.5
RUN conda install -y -c conda-forge python-kubernetes=9.0

RUN mkdir -p /root/saturn/pdc/dask
COPY dask*.py /root/saturn/pdc/dask

ENV PYTHONPATH=/root/saturn

