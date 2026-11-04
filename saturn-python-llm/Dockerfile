ARG SATURNBASE_GPU_IMAGE
FROM ${SATURNBASE_GPU_IMAGE}

RUN sudo apt-get -qq --allow-releaseinfo-change update && \
    sudo apt-get -qq install --yes --no-install-recommends \
    libgl1

COPY environment.yml /tmp/environment.yml
# axolotl 0.16.1 is installed --no-deps so its over-strict transformers==5.5.0
# metadata pin cannot drag transformers 5.x in and break vLLM 0.11 at boot.
# Its transitive deps are declared explicitly in environment.yml. This is a
# separate step because a --no-deps line inside the env.yml pip: block would
# apply to the whole block, suppressing deps for every pip entry.
RUN mamba env update -n saturn --file /tmp/environment.yml && \
    ${CONDA_DIR}/envs/saturn/bin/python -m pip install --no-deps axolotl==0.16.1 && \
    ${CONDA_DIR}/envs/saturn/bin/python -m ipykernel install \
        --name python3 \
        --display-name 'saturn (Python 3)' \
        --prefix=${CONDA_DIR} && \
    ${CONDA_DIR}/bin/conda clean -afy && \
    find ${CONDA_DIR} -type f,l -name '*.pyc' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.a' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.js.map' -delete
RUN echo '' > ${CONDA_DIR}/envs/saturn/conda-meta/history
