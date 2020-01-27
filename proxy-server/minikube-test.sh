#!/bin/bash -e

which docker > /dev/null || (echo "docker is not installed. Exiting." && exit 1)
which minikube > /dev/null || (echo "minikube is not installed. Exiting." && exit 1)

PROXY_AUTH=${PROXY_AUTH:-"false"}
WEB_HOSTNAME="web.localtest.me"

eval $(minikube docker-env)

echo -e "\033[1;92mBuilding proxy image\033[0m"
docker build -t proxy-server:test .

echo -e "\033[1;92mBuilding test server image\033[0m"
(cd test && docker build -t mock-auth-server:test .)

kubectl apply -f minikube-test.yaml
if $PROXY_AUTH; then
    kubectl apply -f minikube-mock-auth.yaml
fi

echo "Waiting for all pods to be ready..."
if $PROXY_AUTH; then
    kubectl wait --for=condition=Ready pod/mock-auth-server
fi
kubectl wait --for=condition=Ready pod/proxy-test

if $PROXY_AUTH; then
    kubectl port-forward svc/mock-auth-server 8888:8888 &
    auth_pid=$!
    sleep 0.1
fi



echo -e "\033[1;92mReady for test\033[0m"

echo "
Open this URL in your browser: http://${WEB_HOSTNAME}/
(you'll need to map web.localtest.me to $(minikube ip) in /etc/hosts if you haven't already)
"

read -p "Press any key to tear down resources."

if $PROXY_AUTH; then
    kill $auth_pid || true # let deletion occur when there was an issue with port-forwarding
fi
kubectl delete -f minikube-test.yaml
