
ARG SATURNBASE_IMAGE
FROM ${SATURNBASE_IMAGE}

COPY postBuild /tmp/postBuild.sh

RUN sudo apt-get install libzmq3-dev

RUN /bin/bash -e -u -o pipefail /tmp/postBuild.sh && \
    sudo rm /tmp/postBuild.sh
