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
    autoconf \
    binutils \
    build-essential \
    ca-certificates \
    clang-15 \
    cmake \
    curl \
    debhelper \
    devscripts \
    dh-make \
    git \
    gnupg2 \
    libcurl4-openssl-dev \
    libssl-dev \
    lsb-release \
    make \
    ninja-build \
    pkg-config \
    postgresql-common \
    python3.11 \
    python3.11-dev \
    python3.11-venv \
    wget \
    ; \
    sh /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
    postgresql-server-dev-all \
    ; \
    apt-get clean

# patch pg_buildext to use multiple processors
COPY patch/make_pg_buildext_parallel.patch /tmp
RUN patch `which pg_buildext` < /tmp/make_pg_buildext_parallel.patch

COPY --from=gobuild /go/bin/* /usr/local/bin/

# add pgxman repo
RUN pgxman update
