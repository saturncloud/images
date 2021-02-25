include .env_deps
export

build_image:
	docker build \
		--no-cache \
		--build-arg JUPYTER_SATURN_VERSION=${JUPYTER_SATURN_VERSION} \
		-t ${IMAGE} \
		.
