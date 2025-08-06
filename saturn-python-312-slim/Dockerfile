FROM python:3.12-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1

RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    git \
    wget \
    > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*

ENV USER=jovyan
ENV HOME=/home/${USER}
ENV UID=1000

RUN useradd -m -u ${UID} ${USER} && \
    echo "${USER} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook

USER ${USER}
WORKDIR ${HOME}

RUN pip install --no-cache-dir \
    ipykernel \
    jupyterlab \
    notebook \
    voila

EXPOSE 8888

ENTRYPOINT []