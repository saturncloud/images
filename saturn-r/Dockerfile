
ARG SATURNBASE_IMAGE
FROM ${SATURNBASE_IMAGE}

COPY environment.yml /tmp/environment.yml
COPY postbuild.sh /tmp/postbuild.sh

# https://stat.ethz.ch/R-manual/R-devel/library/base/html/libPaths.html
ENV R_LIBS=/usr/local/lib/R/

RUN conda env update -n saturn --file /tmp/environment.yml && \
    find ${CONDA_DIR} -type f,l -name '*.pyc' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.a' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.js.map' -delete && \
    /bin/bash -e -u -o pipefail /tmp/postbuild.sh && \
    sudo rm /tmp/postbuild.sh && \
    echo '' > ${CONDA_DIR}/envs/saturn/conda-meta/history
