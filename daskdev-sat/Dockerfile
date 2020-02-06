FROM daskdev/dask:2.8.1

RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends locales > /dev/null && \
    apt-get -y install wget git curl cron unzip && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*
ENV SHELL /bin/bash

RUN conda install -y dask=2.10.1  distributed
RUN conda install -y -c conda-forge kubernetes=1.13.5 dask-kubernetes=0.10.0 python-kubernetes=9.0

#TODO - remove this pdc crap
#TODO - run as non-root user
RUN mkdir -p /root/saturn/pdc/dask
COPY dask_kubernetes_start.py /root/saturn/pdc/dask

ENV PYTHONPATH=/root/saturn
