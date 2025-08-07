include .env_deps
export

build_image:
	docker build \
		--no-cache \
		--build-arg SATURNBASE_GPU_IMAGE=${SATURNBASE_GPU_IMAGE} \
		-t ${IMAGE} \
		.