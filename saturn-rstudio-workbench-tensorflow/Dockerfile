
ARG SATURNBASE_GPU_IMAGE
FROM ${SATURNBASE_GPU_IMAGE}

COPY environment.yml /tmp/environment.yml
COPY postBuild /tmp/postBuild.sh

# https://stat.ethz.ch/R-manual/R-devel/library/base/html/libPaths.html
ENV R_LIBS=/usr/local/lib/R/

RUN mamba env update -n saturn --file /tmp/environment.yml && \
    find ${CONDA_DIR} -type f,l -name '*.pyc' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.a' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.js.map' -delete && \
    /bin/bash -e -u -o pipefail /tmp/postBuild.sh && \
    sudo rm /tmp/postBuild.sh && \
    echo '# cmd: /opt/saturncloud/bin/conda install papermill' > ${CONDA_DIR}/envs/saturn/conda-meta/history
# history command is a hotfix for https://github.com/rstudio/reticulate/issues/1184
