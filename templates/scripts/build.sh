pushd saturnbase
docker build -t saturncloud/saturnbase:{{base_image_version}} .
popd
pushd saturn
docker build -t saturncloud/saturn:{{image_version}} .
popd
pushd saturnbase-gpu
docker build -t saturncloud/saturnbase-gpu:{{base_image_version}} .
popd
pushd saturn-gpu
docker build -t saturncloud/saturn-gpu:{{image_version}} .
popd
pushd saturn-r
docker build -t saturncloud/saturn-r:{{image_version}} .
popd
