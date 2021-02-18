# images

This repository holds code to create Saturn Docker images.

## Default images for customer use

A default image is defined as an image that, upon a fresh customer install, is immediately available to be attached to a Jupyter server or Dask cluster.

All default images should have at least the following packages with appropriate pins, floors, or ceilings. This ensures customers will be able to use Dask, Prefect, and Snowflake in every image.

```yml
name: saturn
channels:
- defaults
- conda-forge
dependencies:
- blas=*=mkl
- bokeh
- dask-ml
- dask
- distributed
- ipykernel
- ipywidgets
- matplotlib
- numpy
- pandas
- pip
- prefect
- pyarrow
- python=3.7
- python-graphviz
- s3fs
- scikit-learn
- scipy
- voila
- xgboost
- pip:
  - dask-saturn
  - prefect-saturn
  - snowflake-connector-python
```

We need to keep images as small as possible, because image size directly impacts instance spinup time.

- saturn: Data analysis, machine learning, and parallel processing with Dask
- saturn-rapids: GPU-acceleration with RAPIDS (GPU instance recommended)
- saturn-tensorflow: Deep learning with tensorflow (GPU instance recommended)
- saturn-pytorch: Deep learning with pytorch (GPU instance recommended)
- examples-cpu: For running examples-cpu project
- examples-gpu: For running examples-gpu project (GPU instance recommended)
- saturn-geospatial: Geospatial IO, analysis and visualization


## Adding a new image definition

Each image is stored in its own subdirectory. That subdirectory should have at least a `Dockerfile` and `.dockerignore`.

**`Dockerfile`**

A script that defines how to build the image.

For complete details on how to write `.dockerignore` files, see [the official docker documentation](https://docs.docker.com/engine/reference/builder/).

**`.dockerignore`**

Similar to `.gitignore`, `.dockerignore` is used to prevent unwanted files from being bundled in an image. For a good explanation of this, see ["Do Not Ignore .dockerignore"](https://codefresh.io/docker-tutorial/not-ignore-dockerignore-2/).

The images in this repository use `.dockerignore` files like this:

```text
*
!app.py
!environment.yml
```

That syntax says "ignore everything EXCEPT `app.py` and `environment.yml`".

For complete details on how to write `.dockerignore` files, see [the docker documentation](https://docs.docker.com/engine/reference/builder/#dockerignore-file).
