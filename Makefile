.PHONY: flake8
flake8:
	flake8 --count --max-line-length 100

.PHONY: check-metadata
check-metadata:
	./.ci/check-standard-images.sh

.PHONY: lint
lint: flake8 check-metadata

.PHONY: sync
sync:
	bash scripts/sync.sh
