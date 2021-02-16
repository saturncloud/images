.PHONY: format
format:
	black --line-length 100 .

.PHONY: flake8
flake8:
	flake8 --count --max-line-length 100

.PHONY: black
black:
	black --check --diff --line-length 100 .

.PHONY: check-metadata
check-metadata:
	./.ci/check-standard-images.sh

.PHONY: lint
lint: flake8 black mypy check-metadata
