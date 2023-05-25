SHELL=/bin/bash -eo pipefail

.PHONY: build
build:
	docker buildx build --platform linux/amd64,linux/arm64 --output out .
