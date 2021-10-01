include .env_deps
export

build_image:
	docker build \
		--build-arg SATURN_R_IMAGE=${SATURN_R_IMAGE} \
		-t ${IMAGE} \
		.
