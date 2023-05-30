SHELL=/bin/bash -eo pipefail

.PHONY: build
build:
	docker buildx build -t ghcr.io/hydradatabase/pgxm/pgvector --platform linux/amd64,linux/arm64 --output out .
