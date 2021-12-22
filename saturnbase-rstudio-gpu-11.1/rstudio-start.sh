#!/bin/bash

echo "Set up environment variables"
while IFS='=' read -r -d '' n v; do
    printf "%s=%s\n" "$n" "$v" | sudo tee -a /usr/lib/R/etc/Renviron >/dev/null
done < <(env -0)

echo "RStudio Server is starting"
rstudio-server start
