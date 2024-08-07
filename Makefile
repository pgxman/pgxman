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
	go build -o $(BIN_DIR)/pgxman ./cmd/pgxman
	go build -o $(BIN_DIR)/pgxman-pack ./cmd/pgxman-pack

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
UBUNTU_NOBLE_IMAGE ?= ghcr.io/pgxman/builder/ubuntu/noble:main
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
		-ubuntu-noble-image $(UBUNTU_NOBLE_IMAGE) \
		-runner-postgres-15-image $(RUNNER_POSTGRES_15_IMAGE) \
		-pgxman-bin $(BIN_DIR)/pgxman_linux_$$(go env GOARCH)

.PHONY: vet
vet:
	docker \
		run \
		--rm \
		-v $(CURDIR):/app \
		-w /app \
		golangci/golangci-lint:v1.56.0 \
		golangci-lint run --timeout 5m -v

.PHONY: docs
docs:
	rm -rf docs/cli docs/man
	docker run --rm -ti -v $(CURDIR):/src -w /src golang:latest bash -c "go run /src/cmd/gendoc/main.go -markdown docs/cli -man docs/man && docs/script/md-to-mdx docs/cli/*.md && rm docs/cli/*.md"

DOCKER_ARGS ?=
.PHONY: docker_build_builder
docker_build_builder:
	docker buildx bake builder \
		-f $(PWD)/dockerfiles/docker-bake.hcl \
		--set builder-debian-bookworm.tags=$(DEBIAN_BOOKWORM_IMAGE) \
		--set builder-ubuntu-jammy.tags=$(UBUNTU_JAMMY_IMAGE) \
		--set builder-ubuntu-noble.tags=$(UBUNTU_NOBLE_IMAGE) \
		--set *.cache-from=type=gha --set *.cache-to=type=gha,mode=max \
		--pull \
		$(DOCKER_ARGS)

.PHONY: docker_load_builder
docker_load_builder: DOCKER_ARGS=--load
docker_load_builder: docker_build_builder

.PHONY: docker_push_builder
docker_push_builder: DOCKER_ARGS=--push
docker_push_builder: docker_build_builder

RUNNER_TARGET ?= runner
RUNNER_POSTGRES_16_IMAGE ?= ghcr.io/pgxman/runner/postgres/16:main
RUNNER_POSTGRES_15_IMAGE ?= ghcr.io/pgxman/runner/postgres/15:main
RUNNER_POSTGRES_14_IMAGE ?= ghcr.io/pgxman/runner/postgres/14:main
RUNNER_POSTGRES_13_IMAGE ?= ghcr.io/pgxman/runner/postgres/13:main
.PHONY: docker_build_runner
docker_build_runner:
	docker buildx bake $(RUNNER_TARGET) \
		-f $(PWD)/dockerfiles/docker-bake.hcl \
		--set runner-postgres-16.tags=$(RUNNER_POSTGRES_16_IMAGE) \
		--set runner-postgres-15.tags=$(RUNNER_POSTGRES_15_IMAGE) \
		--set runner-postgres-14.tags=$(RUNNER_POSTGRES_14_IMAGE) \
		--set runner-postgres-13.tags=$(RUNNER_POSTGRES_13_IMAGE) \
		--set *.cache-from=type=gha --set *.cache-to=type=gha,mode=max \
		--pull \
		$(DOCKER_ARGS)

.PHONY: docker_load_runner
docker_load_runner: DOCKER_ARGS=--load
docker_load_runner: docker_build_runner

.PHONY: docker_push_runner
docker_push_runner: DOCKER_ARGS=--push
docker_push_runner: docker_build_runner

.PHONY: installer_test
installer_test:
	docker build \
		--rm \
		-f $(PWD)/dockerfiles/test/Dockerfile.installer_test \
		.

.PHONY: goreleaser
goreleaser:
	goreleaser release --clean --snapshot --skip=publish
