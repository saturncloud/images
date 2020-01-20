#!/bin/bash -e

which docker > /dev/null || (echo "docker is not installed. Exiting." && exit 1)
which minikube > /dev/null || (echo "minikube is not installed. Exiting." && exit 1)

eval $(minikube docker-env)

echo -e "\033[1;92mBuilding proxy image\033[0m"
docker build -t proxy-server:test .

echo -e "\033[1;92mBuilding test server image\033[0m"
(cd test && docker build -t mock-auth-server:test .)

kubectl apply -f minikube-test.yaml

echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready pod/mock-auth-server

kubectl port-forward svc/mock-auth-server 8888:8888 &
auth_pid=$!
sleep 0.1



echo -e "\033[1;92mReady for test\033[0m"

echo "
Open this URL in your browser: $(minikube service proxy-test --url=true)/resource
"

read -p "Press any key to stop forwarding and tear down resources."

kill $auth_pid || true # let deletion occur when there was an issue with port-forwarding
kubectl delete -f minikube-test.yaml
