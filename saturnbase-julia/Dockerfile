FROM ubuntu:jammy
EXPOSE 8888

ENV JULIA_VERSION=1.7.3
ARG CONDA_VERSION=py39_4.11.0

ENV CONDA_DIR=/opt/saturncloud
ENV CONDA_BIN=${CONDA_DIR}/bin
ENV DEBIAN_FRONTEND=noninteractive

# install Julia packages in /opt/saturncloud/julia instead of ${HOME}
ENV JULIA_DEPOT_PATH=/opt/saturncloud/julia
ENV JULIA_PKGDIR=/opt/saturncloud/julia

RUN apt-get -qq --allow-releaseinfo-change update && \
    apt-get -qq upgrade -y && \
    apt-get -qq install --yes --no-install-recommends \
    awscli \
    dumb-init \
    build-essential \
    bzip2 \
    ca-certificates \
    curl \
    file \
    gettext-base \
    git \
    gnupg \
    htop \
    libnuma-dev \
    locales \
    openssh-server \
    rsync \
    screen \
    ssh \
    sudo \
    unzip \
    wget \
    zip \
    > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /run/sshd && \
    chmod 755 /run/sshd && \
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen

ENV LC_ALL=en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US.UTF-8
ENV SHELL=/bin/bash
ENV NB_USER=jovyan
ENV NB_UID=1000
ARG NB_GID=100

COPY install-miniconda.bash /tmp/install-miniconda.bash
ARG CONDA_VERSION=py39_4.11.0
RUN bash /tmp/install-miniconda.bash

ENV USER=${NB_USER}
ENV HOME=/home/${NB_USER}

RUN adduser --disabled-password \
    --gecos "Default user" \
    --uid ${NB_UID} \
    ${NB_USER} && \
    echo "$NB_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook && \
    chown 1000:1000 -R /opt/saturncloud

COPY profile /etc/profile

USER ${NB_USER}

COPY install-jupyter.bash /tmp/install-jupyter.bash
COPY environment.yml /tmp/environment.yml
RUN bash /tmp/install-jupyter.bash && \
    echo '' > ${CONDA_DIR}/conda-meta/history && \
    ${CONDA_BIN}/conda config --system --add channels conda-forge && \
    ${CONDA_BIN}/conda config --system --set auto_update_conda false && \
    ${CONDA_BIN}/conda config --system --set show_channel_urls true

ENV NB_PYTHON_PREFIX ${CONDA_DIR}/envs/saturn
ENV PATH ${NB_PYTHON_PREFIX}/bin:${CONDA_BIN}:${HOME}/.local/bin:${PATH}


# Copy a script that we will use to correct permissions after running certain commands
COPY fix-permissions /usr/local/bin/fix-permissions
RUN sudo chmod a+rx /usr/local/bin/fix-permissions

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

WORKDIR /tmp

RUN set -x && \
    julia_arch=$(uname -m) && \
    julia_short_arch="${julia_arch}" && \
    if [ "${julia_short_arch}" == "x86_64" ]; then \
    julia_short_arch="x64"; \
    fi; \
    julia_installer="julia-${JULIA_VERSION}-linux-${julia_arch}.tar.gz" && \
    julia_major_minor=$(echo "${JULIA_VERSION}" | cut -d. -f 1,2) && \
    sudo mkdir "/opt/saturncloud/julia-${JULIA_VERSION}" && \
    wget -q "https://julialang-s3.julialang.org/bin/linux/${julia_short_arch}/${julia_major_minor}/${julia_installer}" && \
    sudo tar xzf "${julia_installer}" -C "/opt/saturncloud/julia-${JULIA_VERSION}" --strip-components=1 && \
    rm "${julia_installer}" && \
    sudo ln -fs /opt/saturncloud/julia-*/bin/julia /usr/local/bin/julia && \
    # Show Julia where conda libraries are \
    sudo mkdir /etc/julia && \
    sudo sh -c 'echo "push!(Libdl.DL_LOAD_PATH, \"${CONDA_DIR}/lib\")" >> /etc/julia/juliarc.jl' && \
    # Create JULIA_PKGDIR \
    sudo mkdir "${JULIA_PKGDIR}" && \
    sudo chown "${NB_USER}" "${JULIA_PKGDIR}" && \
    fix-permissions "${JULIA_PKGDIR}" && \
    # Install IJulia as jovyan
    julia -e 'import Pkg; Pkg.update()' && \
    julia -e 'using Pkg; pkg"add IJulia"; pkg"precompile"' && \
    # move kernelspec out of system share
    mv "${HOME}/.local/share/jupyter/kernels/julia"* "${CONDA_DIR}/share/jupyter/kernels/" && \
    chmod -R go+rx "${CONDA_DIR}/share/jupyter" && \
    rm -rf "${HOME}/.local" && \
    fix-permissions "${JULIA_PKGDIR}" "${CONDA_DIR}/share/jupyter" && \
    sudo rm /usr/local/bin/fix-permissions

WORKDIR ${HOME}
