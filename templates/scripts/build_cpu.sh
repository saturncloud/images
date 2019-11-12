docker build -t saturncloud/saturnbase:{{base_image_version}} saturnbase
docker push saturncloud/saturnbase:{{base_image_version}}
docker build -t saturncloud/saturn:{{base_image_version}} saturn
docker push saturncloud/saturn:{{base_image_version}}
