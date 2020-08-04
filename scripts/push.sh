docker tag saturncloud/saturnbase:0.9.3.7 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturnbase:0.9.3.7
docker tag saturncloud/saturn:0.9.3.7 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturn:0.9.3.7
docker push 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturnbase:0.9.3.7
docker push 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturn:0.9.3.7
docker push saturncloud/saturnbase:0.9.3.7
docker push saturncloud/saturn:0.9.3.7

# docker tag saturncloud/saturn-gpu:0.9.3.7 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturn-gpu:0.9.3.7
# docker tag saturncloud/saturnbase-gpu:0.9.3.7 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturnbase-gpu:0.9.3.7
# docker push 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturn-gpu:0.9.3.7
# docker push 369111250314.dkr.ecr.us-east-2.amazonaws.com/saturnbase-gpu:0.9.3.7
# docker push saturncloud/saturn-gpu:0.9.3.7
# docker push saturncloud/saturnbase-gpu:0.9.3.7
