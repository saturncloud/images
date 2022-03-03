include .env_deps
export

build_image:
	docker build \
		--no-cache \
		--build-arg SATURNBASE_IMAGE=${SATURNBASE_IMAGE} \
		-t narobboo/julia-base:0.06 \
		.