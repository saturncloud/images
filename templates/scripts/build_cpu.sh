pushd saturnbase
docker build -t saturncloud/saturnbase:{{base_image_version}} .
popd
docker push saturncloud/saturnbase:{{base_image_version}}
pushd saturn
docker build -t saturncloud/saturn:{{image_version}} .
popd
docker push saturncloud/saturn:{{image_version}}
