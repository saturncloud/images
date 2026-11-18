ARG SATURNBASE_GPU_IMAGE
FROM ${SATURNBASE_GPU_IMAGE}

# Supply-chain cooldown: pip takes only files uploaded at least 7 days ago
# (pip>=26.1, pinned via environment.yml). A build ARG, not ENV, so it applies
# while the image is built and is not set in customers' workspaces.
ARG PIP_UPLOADED_PRIOR_TO=P7D
COPY environment.yml /tmp/environment.yml
RUN mamba env update -n saturn --file /tmp/environment.yml && \
    ${CONDA_DIR}/envs/saturn/bin/python -m ipykernel install \
    --name python3 \
    --display-name 'saturn (Python 3)' \
    --prefix=${CONDA_DIR} && \
    ${CONDA_DIR}/bin/conda clean -afy && \
    find ${CONDA_DIR} -type f,l -name '*.pyc' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.a' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.js.map' -delete
RUN echo '' > ${CONDA_DIR}/envs/saturn/conda-meta/history
