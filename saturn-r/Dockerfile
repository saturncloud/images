
ARG SATURNBASE_IMAGE
FROM ${SATURNBASE_IMAGE}

# Supply-chain cooldown: pip takes only files uploaded at least 7 days ago
# (pip>=26.1, pinned via environment.yml). A build ARG, not ENV, so it applies
# while the image is built and is not set in customers' workspaces.
ARG PIP_UPLOADED_PRIOR_TO=P7D
COPY environment.yml /tmp/environment.yml
COPY postBuild /tmp/postBuild.sh

RUN mamba env update -n saturn --file /tmp/environment.yml && \
    find ${CONDA_DIR} -type f,l -name '*.pyc' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.a' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.js.map' -delete && \
    /bin/bash -e -u -o pipefail /tmp/postBuild.sh && \
    sudo rm /tmp/postBuild.sh && \
    echo '# cmd: /opt/saturncloud/bin/conda create -n saturn' > ${CONDA_DIR}/envs/saturn/conda-meta/history

# history command is a hotfix for https://github.com/rstudio/reticulate/issues/1184
