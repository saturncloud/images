cpu_image:
	bash scripts/build_cpu.sh

.PHONY: format
format:
	black --line-length 100 .

.PHONY: lint
lint:
	flake8 --count --max-line-length 100
	black --check --diff --line-length 100 .
	mypy --ignore-missing-imports .
