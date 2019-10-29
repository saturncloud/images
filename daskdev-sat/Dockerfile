FROM daskdev/dask:latest

RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends locales > /dev/null && \
    apt-get -y install wget git bzip2 curl python-pip cron unzip && \
    apt-get -y install mc  && \
    apt-get -y install openssh-server  && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*
ENV SHELL /bin/bash

RUN conda update dask
RUN conda update distributed
RUN conda install boto3
RUN conda install s3fs
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.14.0/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl


RUN conda install -y -c conda-forge dask-kubernetes
RUN conda install -c https://conda.saturncloud.io/pkgs sutils=0.5.1

RUN mkdir /root/saturn
COPY saturn /root/saturn

## temp!
RUN mkdir /root/.minikube
COPY .kube/config /root/.kube/config
COPY client.* /root/.minikube/
COPY ca.crt /root/.minikube/


