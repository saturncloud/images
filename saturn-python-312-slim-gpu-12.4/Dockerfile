FROM nvidia/cuda:12.4.0-cudnn8-runtime-ubuntu22.04

ENV CONDA_OVERRIDE_CUDA=12.4
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1

RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    git \
    python3-pip \
    python3-dev \
    python3-venv \
    wget \
    > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*

RUN ln -s /usr/bin/python3 /usr/bin/python

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
    voila \
    torch \
    torchvision \
    torchaudio

EXPOSE 8888

ENTRYPOINT []