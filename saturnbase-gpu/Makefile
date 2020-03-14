include .env_deps
export

build_image:
	docker build --build-arg JUPYTER_SATURN_VERSION=${JUPYTER_SATURN_VERSION} --build-arg SUTILS_VERSION=${SUTILS_VERSION} -t ${IMAGE} .
