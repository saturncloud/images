pushd saturnbase
docker build -t saturncloud/saturnbase:0.9.3.7 .
popd
pushd saturn
docker build -t saturncloud/saturn:0.9.3.7 .
popd
pushd saturnbase-gpu
docker build -t saturncloud/saturnbase-gpu:0.9.3.7 .
popd
pushd saturn-gpu
docker build -t saturncloud/saturn-gpu:0.9.3.7 .
popd
