
ARG SATURNBASE_IMAGE
FROM ${SATURNBASE_IMAGE}

COPY environment.yml /tmp/environment.yml
COPY postBuild /tmp/postBuild.sh
COPY r-jupyter-kernel /tmp/r-jupyter-kernel.sh 


RUN conda env update -n saturn --file /tmp/environment.yml && \
    find ${CONDA_DIR} -type f,l -name '*.pyc' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.a' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.js.map' -delete
    
RUN /bin/bash /tmp/postBuild.sh && \
    chmod 755 /tmp/r-jupyter-kernel.sh

RUN echo '' > ${CONDA_DIR}/envs/saturn/conda-meta/history

ENTRYPOINT ["r-jupyter-kernel.sh"]
