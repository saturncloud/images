# Saturn Base RStudio GPU 11.1

## Description
A base image for Saturn RStudio GPU images, built on CUDA version 11.1 using [rocker/ml](https://github.com/rocker-org/rocker-versioned2) as the starting point. The rocker/ml image is copyright of the rocker project. This image contains the packages necessary to run R, RStudio, Python, JupyterLab, and Dask and sets up environmental variables for Reticulate support. Also installs a few libraries that are useful for Markdown support.
<hr>

**OS**: Ubuntu

|**Python Packages**|**R Packages**|
|---|---|
|[environment.yml](environment.yml)|jquerylib</br>markdown</br>rmarkdown</br>tinytex|