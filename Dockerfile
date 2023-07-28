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
    ./cmd/...

FROM postgres:15-bookworm

ARG DEBIAN_FRONTEND=noninteractive

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    gnupg2 \
    postgresql-common \
    ; \
    sh /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
    autoconf \
    build-essential \
    ca-certificates \
    cmake \
    curl \
    debhelper \
    devscripts \
    dh-make \
    gcc \
    libcurl4-openssl-dev \
    libssl-dev \
    lsb-release \
    make \
    ninja-build \
    pkg-config \
    postgresql-server-dev-all \
    python3.11 \
    python3.11-dev \
    python3.11-venv \
    wget \
    ; \
    rm -rf /var/lib/apt/lists/*

# patch pg_buildext to use multiple processors
COPY patch/make_pg_buildext_parallel.patch /tmp
RUN patch `which pg_buildext` < /tmp/make_pg_buildext_parallel.patch

COPY --from=gobuild /go/bin/* /usr/local/bin/

# add pgxman repo
RUN pgxman update
