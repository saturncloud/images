#!/bin/bash
set -eo pipefail

usage() {
    echo '
    Usage:
        Commands:
            build: Build image, push to local registry, and set image on saturn-auth-proxy deployment(s)
        Options:
            -n, --namespace: Namespace(s) to update (default: "main-namespace anon-namespace"
            --tail: Tail logs after deploying new image. If multiple namepsaces are specified by namespace param (e.g. default),
                    you may pass a single namespace to tail with this param.

    Examples:
        # Set new image only in main, and tail logs
        ./local.sh build -n main-namespace --tail

        # Set new image in both namesapces, and tail the proxy in main
        ./local.sh build --tail main-namespace
'
}

NAMESPACES="main-namespace anon-namespace"
TAIL=""
TAIL_NS=""
while [ "$1" ]; do
    case $1 in
        build )
            COMMAND=$1
            ;;
        -n | --namespace )
            shift
            NAMESPACES="$1"
            ;;
        --tail )
            if [[ $2 != -* ]]; then
                shift
                TAIL_NS=$1
            fi
            TAIL=true
            ;;
        * )
            echo "Error: Unkown parameter \"$1\""
            exit 1
            ;;
    esac
    shift
done

header() {
    echo -e "\n${@}"
    DASHES=$(( $(echo "${@}" | wc -c) * 2 ))
    for i in $(seq $DASHES); do
        echo -n "-"
    done
    echo
}

if [ ! "$COMMAND" ]; then
    echo "Error: Missing <COMMAND>"
    exit 1
fi

NUM_NAMESPACES=$(echo $NAMESPACES | wc -w)
if [ "$TAIL" ] && [ ! "$TAIL_NS" ] && (( $NUM_NAMESPACES > 1 )); then
    echo "Error: Unable to tail proxy in multiple namespaces. Specify a namespace."
elif [ ! "$TAIL_NS" ] && (( $NUM_NAMESPACES == 1 )); then
    TAIL_NS=$NAMESPACES
fi

DIR=$(dirname $0)

IMAGE=localhost:32000/proxy-server
TAG_PREFIX=test

LATEST_TAG=$(docker images | grep "$IMAGE" |  awk 'NR==1{print $2}')
TAG="$TAG_PREFIX$(($(echo $LATEST_TAG | sed 's/'$TAG_PREFIX'//') + 1))"

header Build
docker build -t $IMAGE:$TAG $DIR

OLD_ID=$(docker inspect --format {{.Id}} $IMAGE:$LATEST_TAG)
NEW_ID=$(docker inspect --format {{.Id}} $IMAGE:$TAG)

if [ "$OLD_ID" == "$NEW_ID" ]; then
    echo -e "\nNo changes from previous build"
    docker rmi $IMAGE:$TAG
    TAG=$LATEST_TAG
else
    header Push
    docker push $IMAGE:$TAG
fi

header Rollout
for NS in $NAMESPACES; do
    echo "$NS"
    if kubectl -n $NS get deployment saturn-auth-proxy &>/dev/null; then
        kubectl -n $NS set image deployment/saturn-auth-proxy --record proxy=$IMAGE:$TAG
    else
        echo "No deployment found"
    fi
done

if [ "$TAIL" ]; then
    header Wait
    kubectl -n $TAIL_NS rollout status deployment saturn-auth-proxy
    header Logs
    kubectl -n $TAIL_NS logs -f deployment/saturn-auth-proxy
fi