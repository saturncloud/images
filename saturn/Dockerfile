ARG SATURNBASE_IMAGE=saturncloud/saturnbase:2020.03.09.dev.1
FROM ${SATURNBASE_IMAGE}

COPY environment.yml /tmp/environment.yml
RUN conda env update -n saturn --file /tmp/environment.yml && \
    ${CONDA_DIR}/envs/saturn/bin/python -m ipykernel install --name python3 --prefix=${CONDA_DIR} && \
    ${CONDA_DIR}/bin/conda clean -afy && \
    find ${CONDA_DIR} -type f,l -name '*.pyc' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.a' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.js.map' -delete
RUN echo '' > ${CONDA_DIR}/envs/saturn/conda-meta/history
RUN ${CONDA_DIR}/bin/conda install -n root -c saturncloud dask-labextension "dask-saturn=0.0.2" "dask=2.12.0" "distributed=2.12.0"

RUN ${CONDA_DIR}/bin/jupyter labextension install dask-labextension
RUN ${CONDA_DIR}/bin/jupyter serverextension enable --py dask_labextension --sys-prefix

