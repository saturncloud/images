ARG SATURNBASE_GPU_IMAGE
FROM ${SATURNBASE_GPU_IMAGE}

RUN sudo apt-get -qq --allow-releaseinfo-change update && \
    sudo apt-get -qq install --yes --no-install-recommends \
    libgl1

COPY environment.yml /tmp/environment.yml
# Unlike the serving image (saturn-python-llm), axolotl is installed WITH its
# deps here: this image has no vLLM, so there is no transformers<5 constraint to
# protect, and axolotl 0.16.1 needs the transformers 5.x API at runtime. Letting
# it resolve its own tree (transformers 5.5, datasets 4.5, trl 0.29, hf-hub>=1,
# accelerate 1.13, ...) is what makes training actually work. axolotl is declared
# in environment.yml's pip: block with its extras, so a normal env update pulls
# everything; no separate --no-deps step.
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
