pushd saturnbase
docker build -t saturncloud/saturnbase:0.7.4 .
popd
pushd saturnbase-gpu
docker build -t saturncloud/saturnbase-gpu:0.7.4 .
popd
pushd saturn
docker build -t saturncloud/saturn:0.7.4 .
popd
pushd saturn-gpu
docker build -t saturncloud/saturn-gpu:0.7.4 .
popd
pushd saturn-r
docker build -t saturncloud/saturn-r:0.7.4 .
popd
# docker push saturncloud/saturn:0.7.1
# docker push saturncloud/saturnbase:0.7.0
# docker push saturncloud/saturn-gpu:0.7.1
# docker push saturncloud/saturnbase-gpu:0.7.0
