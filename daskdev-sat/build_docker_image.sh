#!/bin/bash

DASKDEV_SAT_VERSION=1.02
echo "Building daskdev-sat version $DASKDEV_SAT_VERSION" 

mkdir -p saturn/pdc/dask
cp     ../../saturn/pdc/dask/*.py  ./saturn/pdc/dask/

sudo docker build -t saturncloud/daskdev-sat:$DASKDEV_SAT_VERSION .
sudo docker push saturncloud/daskdev-sat:$DASKDEV_SAT_VERSION
#sudo docker tag saturncloud/daskdev-sat:$DASKDEV_SAT_VERSION saturncloud/daskdev-sat:latest

rm -r ./saturn/
