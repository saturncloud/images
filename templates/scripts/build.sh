pushd saturnbase
docker build -t saturncloud/saturnbase:{{base_image_version}} .
popd

pushd saturnbase-gpu
docker build -t saturncloud/saturnbase-gpu:{{base_image_version}} .
popd

pushd saturn
docker build -t saturncloud/saturn:{{version}} .
popd

pushd saturn-gpu
docker build -t saturncloud/saturn-gpu:{{version}} .
popd

pushd saturn-r
docker build -t saturncloud/saturn-r:{{version}} .
popd
