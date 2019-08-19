pushd saturnbase
docker build -t saturncloud/saturnbase:0.8.3 .
popd

pushd saturnbase-gpu
docker build -t saturncloud/saturnbase-gpu:0.8.3 .
popd

pushd saturn
docker build -t saturncloud/saturn: .
popd

pushd saturn-gpu
docker build -t saturncloud/saturn-gpu: .
popd

pushd saturn-r
docker build -t saturncloud/saturn-r: .
popd