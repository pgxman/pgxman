SHELL=/bin/bash -eo pipefail

REPO ?= ghcr.io/hydradatabase/pgxm/pgvector
PLATFORM ?= linux/amd64,linux/arm64

.PHONY: build
build:
	docker buildx build -t $(REPO) --platform $(PLATFORM) --output out .

.PHONY: debug_build
debug_build:
	docker buildx build -t $(REPO) --target build --load .
