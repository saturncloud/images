docker tag saturncloud/saturnbase:{{base_image_version}} 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturnbase:{{base_image_version}}
docker tag saturncloud/saturn:{{image_version}} 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturn:{{image_version}}
docker push 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturnbase:{{base_image_version}}
docker push 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturn:{{image_version}}
docker push saturncloud/saturnbase:{{base_image_version}}
docker push saturncloud/saturn:{{image_version}}

# docker tag saturncloud/saturn-gpu:{{image_version}} 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturn-gpu:{{image_version}}
# docker tag saturncloud/saturnbase-gpu:{{base_image_version}} 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturnbase-gpu:{{base_image_version}}
# docker push 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturn-gpu:{{image_version}}
# docker push 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturnbase-gpu:{{base_image_version}}
# docker push saturncloud/saturn-gpu:{{image_version}}
# docker push saturncloud/saturnbase-gpu:{{base_image_version}}
