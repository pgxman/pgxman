SHELL=/bin/bash -eo pipefail

BIN_DIR ?= $(CURDIR)/bin
export PATH := $(BIN_DIR):$(PATH)
.PHONY: tools
tools:
	# goreleaser
	GOBIN=$(BIN_DIR) go install github.com/goreleaser/goreleaser@latest

.PHONY: build
build:
	GOBIN=$(BIN_DIR) go build ./cmd/...

.PHONY: install
install:
	go install ./cmd/pgxman
	go install ./cmd/pgxman-pack

GO_TEST_FLAGS ?=
.PHONY: test
test:
	go test $$(go list ./... | grep -v e2etest) $(GO_TEST_FLAGS) -count=1 -race -v

E2ETEST_BUILD_IMAGE ?= ghcr.io/pgxman/builder:main
.PHONY: e2etest
e2etest:
	GOOS=linux GOARCH=$$(go env GOARCH) go build -o $(BIN_DIR)/pgxman_linux_$$(go env GOARCH) ./cmd/pgxman
	go test ./internal/e2etest/ $(GO_TEST_FLAGS) -count=1 -race -v -e2e -build-image $(E2ETEST_BUILD_IMAGE) -pgxman-bin $(BIN_DIR)/pgxman_linux_$$(go env GOARCH)

.PHONY: vet
vet:
	docker \
		run \
		--rm \
		-v $(CURDIR):/app \
		-w /app \
		golangci/golangci-lint:latest \
		golangci-lint run --timeout 5m -v

REPO ?= ghcr.io/pgxman/builder
.PHONY: docker_build
docker_build:
	docker buildx build -t $(REPO) --load --pull .

.PHONY: docker_push
docker_push:
	docker buildx build -t $(REPO) --platform linux/amd64,linux/arm64 --push --pull .

.PHONY: goreleaser
goreleaser:
	goreleaser release --clean --snapshot --skip-publish
