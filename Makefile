SHELL=/bin/bash -eo pipefail

BIN_DIR ?= $(CURDIR)/bin
export PATH := $(BIN_DIR):$(PATH)

all: gen build

.PHONY: tools
tools:
	rm -rf $(BIN_DIR) && mkdir -p $(BIN_DIR)
	# go tools
	GOBIN=$(TOOLS_DIR) go generate -tags tools tools.go
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

.PHONY: gen
gen:
	go generate ./...

.PHONY: cp_registry_spec
cp_registry_spec:
	cp ../registry/oapi/oapi.yaml ./oapi/oapi.yaml

DEBIAN_BOOKWORM_IMAGE ?= ghcr.io/pgxman/builder/debian/bookworm:main
UBUNTU_JAMMY_IMAGE ?= ghcr.io/pgxman/builder/ubuntu/jammy:main
.PHONY: e2etest
e2etest:
	GOOS=linux GOARCH=$$(go env GOARCH) go build -o $(BIN_DIR)/pgxman_linux_$$(go env GOARCH) ./cmd/pgxman
	go test ./internal/e2etest/ \
		$(GO_TEST_FLAGS) \
		-count=1 -race \
		-v \
		-e2e \
		-debian-bookworm-image $(DEBIAN_BOOKWORM_IMAGE) \
		-ubuntu-jammy-image $(UBUNTU_JAMMY_IMAGE) \
		-pgxman-bin $(BIN_DIR)/pgxman_linux_$$(go env GOARCH)

.PHONY: vet
vet:
	docker \
		run \
		--rm \
		-v $(CURDIR):/app \
		-w /app \
		golangci/golangci-lint:v1.54.2 \
		golangci-lint run --timeout 5m -v

DOCKER_ARGS ?=
.PHONY: docker_build
docker_build:
	docker buildx bake \
		-f $(PWD)/dockerfiles/builder/docker-bake.hcl \
		--set debian-bookworm.tags=$(DEBIAN_BOOKWORM_IMAGE) \
		--set ubuntu-jammy.tags=$(UBUNTU_JAMMY_IMAGE) \
		--pull \
		$(DOCKER_ARGS)

.PHONY: docker_load
docker_load: DOCKER_ARGS=--load
docker_load: docker_build

.PHONY: docker_push
docker_push: DOCKER_ARGS=--push
docker_push: docker_build

.PHONY: installer_test
installer_test: goreleaser
	docker build \
		--rm \
		-f $(PWD)/docker/Dockerfile.installer_test \
		.

.PHONY: goreleaser
goreleaser:
	goreleaser release --clean --snapshot --skip=publish
