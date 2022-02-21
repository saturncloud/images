
ARG SATURNBASE_IMAGE
FROM ${SATURNBASE_IMAGE}

COPY postBuild /tmp/postBuild.sh

# https://stat.ethz.ch/R-manual/R-devel/library/base/html/libPaths.html
ENV R_LIBS=/usr/local/lib/R/

RUN /bin/bash -e -u -o pipefail /tmp/postBuild.sh && \
    sudo rm /tmp/postBuild.sh
