#!/bin/bash

echo "Set up environment variables"
env | sudo tee -a /usr/local/lib/R/etc/Renviron > /dev/null

echo "RStudio Server is starting"
rstudio-server start
