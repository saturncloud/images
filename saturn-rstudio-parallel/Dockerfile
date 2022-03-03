
ARG SATURNBASE_IMAGE
FROM ${SATURNBASE_IMAGE}

COPY postBuild /tmp/postBuild.sh

## Otherwise timedatectl will get called which leads to 'no systemd' inside Docker
ENV TZ UTC

RUN sudo apt-get install libzmq3-dev

RUN /bin/bash -e -u -o pipefail /tmp/postBuild.sh && \
    sudo rm /tmp/postBuild.sh
