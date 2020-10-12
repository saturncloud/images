.PHONY: format
format:
	black --line-length 100 .

.PHONY: lint
lint: flake8 black mypy

.PHONY: flake8
flake8:
	flake8 --count --max-line-length 100

.PHONY: black
black:
	black --check --diff --line-length 100 .

.PHONY: mypy
mypy:
	# mypy errors out if passed a directory with no python files in it
	mypy --ignore-missing-imports daskdev-sat
