SHELL=/bin/bash -eo pipefail

.PHONY: build
build:
	go build -o build/pgxman ./cmd/pgxman
	go build -o build/pgxman-pack ./cmd/pgxman-pack

.PHONY: install
install:
	go install ./cmd/pgxman
	go install ./cmd/pgxman-pack

REPO ?= ghcr.io/hydradatabase/pgxm/builder
.PHONY: docker_build
docker_build:
	docker buildx build -t $(REPO) --load .

.PHONY: docker_push
docker_push:
	docker buildx build -t $(REPO) --platform linux/amd64,linux/arm64 --push .

EXAMPLE_REPO ?= ghcr.io/hydradatabase/pgxm/example
.PHONY: example
example:
	docker buildx build -t $(EXAMPLE_REPO) --load .
