FROM saturncloud/saturnbase:{{base_image_version}}

COPY environment.yml /tmp/environment.yml
RUN conda env update -n saturn --file /tmp/environment.yml
RUN echo '' > ${CONDA_DIR}/envs/saturn/conda-meta/history
RUN ${CONDA_DIR}/envs/saturn/bin/python -m ipykernel install --name python3 --prefix=${CONDA_DIR}
RUN ${CONDA_DIR}/bin/conda clean --all

{% if jsaturn_version == 'local' %}
COPY --chown=jovyan jupyterlab_saturn ${CONDA_DIR}/jsaturn_ext
RUN cd ${CONDA_DIR}/jsaturn_ext && \
    ${CONDA_DIR}/bin/npm run build && \
    ${CONDA_DIR}/bin/jupyter labextension install
RUN rm -rf ${CONDA_DIR}/jsaturn_ext
{% endif %}
