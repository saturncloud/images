#!/bin/bash

echo -e "R_LIBS_USER=${R_LIBS}\nR_LIBS_SITE=${R_LIBS}" > ~/.Renviron

echo "RStudio Server is starting"
rstudio-server start
