SHELL=/bin/bash -eo pipefail


DOCKER_REPO ?= ghcr.io/hydradatabase/pgvector

.PHONY: build
build:
	docker buildx build -t $(DOCKER_REPO) --output out .
