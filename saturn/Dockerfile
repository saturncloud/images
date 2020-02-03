ARG SATURNBASE_IMAGE=saturncloud/saturnbase:0.9.3.7
FROM ${SATURNBASE_IMAGE}

COPY environment.yml /tmp/environment.yml
RUN conda env update -n saturn --file /tmp/environment.yml
RUN echo '' > ${CONDA_DIR}/envs/saturn/conda-meta/history
RUN ${CONDA_DIR}/envs/saturn/bin/python -m ipykernel install --name python3 --prefix=${CONDA_DIR}
RUN ${CONDA_DIR}/bin/conda clean --all
