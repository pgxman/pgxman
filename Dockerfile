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

FROM ubuntu:22.04

ARG POSTGRES_VERSION=15
ARG DEBIAN_FRONTEND=noninteractive

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
    gnupg2 \
    postgresql-common \
    ; \
    sh /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y; \
    apt-get update; \
    apt-get upgrade -y; \
    apt-get install -y  \
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
COPY patch/make_pg_buildext_parallel.patch /
RUN patch `which pg_buildext` < /make_pg_buildext_parallel.patch

COPY --from=gobuild /go/bin/* /usr/local/bin/
