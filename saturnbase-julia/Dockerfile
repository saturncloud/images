FROM debian:bullseye-slim
EXPOSE 8888

ENV JULIA_VERSION=1.7.2
ARG CONDA_VERSION=py39_4.10.3

ENV APP_BASE=/srv
ENV CONDA_DIR=/srv/conda
ENV CONDA_BIN=${CONDA_DIR}/bin
ENV DEBIAN_FRONTEND=noninteractive

# install Julia packages in /srv/julia instead of ${HOME}
ENV JULIA_DEPOT_PATH=/srv/julia
ENV JULIA_PKGDIR=/srv/julia

RUN apt-get -qq --allow-releaseinfo-change update && \
    apt-get -qq upgrade -y && \
    apt-get -qq install --yes --no-install-recommends \
    awscli \
    build-essential \
    bzip2 \
    ca-certificates \
    curl \
    gettext-base \
    git \
    gnupg \
    htop \
    libzmq3-dev \
    locales \
    openssh-client \
    openssh-server \
    procps \
    rsync \
    screen \
    sudo \
    tar \
    wget \
    > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /run/sshd && \
    chmod 755 /run/sshd && \
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen

ENV PATH /opt/conda/bin:$PATH

CMD [ "/bin/bash" ]

# Based on https://github.com/ContinuumIO/docker-images/blob/4d7798c0ea2463d9c4057d8eaee876102eecbf86/miniconda3/debian/Dockerfile
RUN set -x && \
    MINICONDA_URL="https://repo.anaconda.com/miniconda/Miniconda3-${CONDA_VERSION}-Linux-x86_64.sh"; \
    SHA256SUM="1ea2f885b4dbc3098662845560bc64271eb17085387a70c2ba3f29fff6f8d52f"; \
    wget "${MINICONDA_URL}" -O miniconda.sh -q && \
    echo "${SHA256SUM} miniconda.sh" > shasum && \
    if [ "${CONDA_VERSION}" != "latest" ]; then sha256sum --check --status shasum; fi && \
    mkdir -p /opt && \
    sh miniconda.sh -b -p /opt/conda && \
    rm miniconda.sh shasum && \
    ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    echo ". /opt/conda/etc/profile.d/conda.sh" >> ~/.bashrc && \
    echo "conda activate base" >> ~/.bashrc && \
    find /opt/conda/ -follow -type f -name '*.a' -delete && \
    find /opt/conda/ -follow -type f -name '*.js.map' -delete && \
    /opt/conda/bin/conda clean -afy && \
    ln -s /opt/conda /srv/conda && \
    chown -R 1000:1000 /opt/conda && \
    chown -R 1000:1000 /srv

ENV LC_ALL=en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US.UTF-8
ENV SHELL=/bin/bash
ENV NB_USER=jovyan
ENV NB_UID=1000
ARG NB_GID=100
ENV USER=${NB_USER}
ENV HOME=/home/${NB_USER}

RUN adduser --disabled-password \
    --gecos "Default user" \
    --uid ${NB_UID} \
    ${NB_USER} && \
    echo "$NB_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/notebook
COPY profile /etc/profile
USER ${NB_USER}

COPY install-jupyter.bash /tmp/install-jupyter.bash
COPY environment.yml /tmp/environment.yml
RUN bash /tmp/install-jupyter.bash && \
    echo '' > ${CONDA_DIR}/conda-meta/history && \
    ${CONDA_BIN}/conda config --system --add channels conda-forge && \
    ${CONDA_BIN}/conda config --system --set auto_update_conda false && \
    ${CONDA_BIN}/conda config --system --set show_channel_urls true
ENV NPM_DIR ${APP_BASE}/npm
ENV NB_PREFIX ${CONDA_DIR}/envs/saturn
ENV PATH ${NB_PREFIX}/bin:${CONDA_BIN}:${NPM_DIR}/bin:${HOME}/.local/bin:${PATH}

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
    sudo mkdir "/srv/julia-${JULIA_VERSION}" && \
    wget -q "https://julialang-s3.julialang.org/bin/linux/${julia_short_arch}/${julia_major_minor}/${julia_installer}" && \
    sudo tar xzf "${julia_installer}" -C "/srv/julia-${JULIA_VERSION}" --strip-components=1 && \
    rm "${julia_installer}" && \
    sudo ln -fs /srv/julia-*/bin/julia /usr/local/bin/julia && \
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
