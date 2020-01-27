#!/bin/bash

# This is our image tag. The tag is <dask_verison>[.<added revision>], so with dask ver 2.8.1  our tags will be 2.8.1, 2.8.1.1, 2.8.1.2, ...
# With dask 2.9  we will have 2.9, 2.9.1, 2.9.2 .....
DASKDEV_SAT_VERSION=2.8.1
echo "Building daskdev-sat version $DASKDEV_SAT_VERSION" 

sudo docker build -t saturncloud/daskdev-sat:$DASKDEV_SAT_VERSION .
sudo docker push saturncloud/daskdev-sat:$DASKDEV_SAT_VERSION
#sudo docker tag saturncloud/daskdev-sat:$DASKDEV_SAT_VERSION saturncloud/daskdev-sat:latest
