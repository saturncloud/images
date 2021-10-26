#!/bin/bash
set -eo pipefail

usage() {
    echo -e "\nUsage: ./build.sh -n <IMAGE_NAME>

    Builds images from subdirectories of parent dir, and splits their conda env across multiple
    layers for more efficient pulls (better parallelization). At minimum, each image dir must
    have an environment.yml file with its conda environment. Image dirs may also contain
    a '.env' file to set ENVs in the Dockerfile, and 'postbuild.sh' to run additional steps
    after building and splitting the conda environment.

    Args:
        -n, --name: Image directory name (required)
        -i, --image-tag: Fully formatted image:tag (default: from .env_deps in image dir)
        -b, --base-image: Base image to build from (default: from .env_deps in image dir)
        -l, --layers: Number of layers to split the conda env into (default: 5)

        --dry-run: Print generated dockerfile and build command, but don't build
        -h, --help: This ;P

    Example:
        ./build.sh -n saturn -i saturncloud/saturn:2021.09.20-example
    "
}

NUM_LAYERS=5
while [ "$1" ]; do
    case $1 in
        -n | --name )
            shift
            IMAGE_NAME=$1
            ;;
        -b | --base-image )
            shift
            BASE_IMAGE=$1
            ;;
        -i | --image-tag )
            shift
            IMAGE=$1
            ;;
        -l | --layers )
            shift
            NUM_LAYERS=$1
            ;;
        --dry-run )
            DRY_RUN=true
            ;;
        -h | --help )
            usage
            exit 0
            ;;
        * )

            echo "Error: Unknown arg ${1}"
            usage
            exit 1
            ;;
    esac
    shift
done

if [ ! "$IMAGE_NAME" ]; then
    echo "Error: IMAGE_NAME is required"
    usage
    exit 1
fi

BASE_DIR=$(realpath $(dirname $0))
IMAGE_DIR=${BASE_DIR}/${IMAGE_NAME}

# Validate image dir
if [ ! -d $IMAGE_DIR ]; then
    echo "Error: No directory $IMAGE_DIR found"
    exit 1
elif [ ! -f $IMAGE_DIR/environment.yml ]; then
    echo "Error: Unable to find conda env $IMAGE_DIR/environment.yml"
    exit 1
elif [ ! -f $IMAGE_DIR/.env_deps ] && ([ ! "${IMAGE}" ] || [ ! "${BASE_IMAGE}" ]); then
    echo "Error: Unable to find $IMAGE_DIR/.env_deps, and no IMAGE or BASE_IMAGE specified"
    exit 1
else
    # Get IMAGE and/or BASE_IMAGE from .env_deps
    if [ ! "${IMAGE}" ]; then
        IMAGE=$(cat $IMAGE_DIR/.env_deps | grep ^IMAGE= | sed 's/IMAGE=//')
    fi
    if [ ! "${BASE_IMAGE}" ]; then
        BASE_IMAGE=$(cat $IMAGE_DIR/.env_deps | grep ^SATURNBASE.*IMAGE= | sed 's/.*IMAGE=//')
    fi
fi

# Validate image and base image values have been set
REQUIRED_ARGS="IMAGE BASE_IMAGE"
for ARG in $REQUIRED_ARGS; do
    if [ ! "${!ARG}" ]; then
        echo "Error: $ARG is required"
        usage
        exit 1
    fi
done

render-dockerfile() {
    local CONDA_ENV_FILE
    local COPY_SPLIT_ENV
    local ENVS
    local POSTBUILD

    # COPY commands
    CONDA_ENV_FILE="COPY ${IMAGE_NAME}/environment.yml /tmp/environment.yml"
    for i in $(seq 0 $(($NUM_LAYERS - 1))); do
        COPY_ENV+="COPY --from=install /data/split/${i}/ /\n"
    done

    # ENV command
    if [ -f ${IMAGE_DIR}/.env ]; then
        ENVS="ENV $(cat ${IMAGE_DIR}/.env | grep -v "^#")"
    fi

    # COPY+RUN postbuild.sh
    if [ -f ${IMAGE_DIR}/postbuild.sh ]; then
        POSTBUILD="COPY ${IMAGE_NAME}/postbuild.sh /tmp/postbuild.sh\nRUN /tmp/postbuild.sh \&\& sudo rm /tmp/postbuild.sh"
    fi

    # Render from template
    cat shared/Dockerfile.template | sed "
        s|#{{BASE_IMAGE}}|${BASE_IMAGE}|;
        s|#{{CONDA_ENV_FILE}}|${CONDA_ENV_FILE}|;
        s|#{{COPY_ENV}}|${COPY_ENV}|;
        s|#{{NUM_LAYERS}}|${NUM_LAYERS}|;
        s|#{{ENVS}}|${ENVS}|g;
        s|#{{POSTBUILD}}|${POSTBUILD}|;
    "
}

BUILD_CMD="docker build -t ${IMAGE} -f ${IMAGE_DIR}/Dockerfile.tmp ${BASE_DIR}"
if [ "$DRY_RUN" == "true" ]; then
    echo "Dockerfile:"
    echo "-----------"
    render-dockerfile
    echo
    echo "Build:"
    echo "------"
    echo ${BUILD_CMD}
else
    render-dockerfile > ${IMAGE_DIR}/Dockerfile.tmp
    $BUILD_CMD
fi
