SHELL=/bin/bash -eo pipefail

.PHONY: build
build:
	go build -o build/pgxm ./cmd/pgxm

EXAMPLE_REPO ?= ghcr.io/hydradatabase/pgxm/example
.PHONY: example
example:
	docker buildx build -t $(EXAMPLE_REPO) --load .
