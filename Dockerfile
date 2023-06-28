# syntax=docker/dockerfile:1

FROM golang:latest as gobuild

ARG TARGETOS TARGETARCH
ARG BUILD_VERSION

WORKDIR /src
ENV CGO_ENABLED=0

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go install \
    -ldflags "-s -w -X github.com/pgxman/pgxman/pgxm.Version=$BUILD_VERSION" \
    ./cmd/pgxman-pack/...

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go install \
    -ldflags "-s -w -X github.com/pgxman/pgxman/pgxm.Version=$BUILD_VERSION" \
    ./cmd/pgxman/...

FROM postgres:15-bookworm
ARG POSTGRES_VERSION=15

ARG DEBIAN_FRONTEND=noninteractive

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
    gnupg2 \
    build-essential \
    ca-certificates \
    debhelper \
    devscripts \
    dh-make \
    lsb-release \
    git \
    curl \
    gcc \
    make \
    libssl-dev \
    autoconf \
    pkg-config \
    postgresql-server-dev-all \
    postgresql-server-dev-${POSTGRES_VERSION} \
    ; \
    rm -rf /var/lib/apt/lists/*

# patch pg_buildext to use multiple processors
COPY patch/make_pg_buildext_parallel.patch /tmp
RUN patch `which pg_buildext` < /tmp/make_pg_buildext_parallel.patch

COPY --from=gobuild /go/bin/* /usr/local/bin/
