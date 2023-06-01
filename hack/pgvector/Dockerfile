# syntax=docker/dockerfile:1

FROM ubuntu:22.04 as base

ARG POSTGRES_VERSION=14
ARG DEBIAN_FRONTEND=noninteractive

RUN set -eux; \
  apt-get update; \
  apt-get install -y --no-install-recommends \
  gnupg \
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
  postgresql-${POSTGRES_VERSION} \
  postgresql-server-dev-all \
  postgresql-server-dev-${POSTGRES_VERSION} \
  ; \
  rm -rf /var/lib/apt/lists/*

FROM base as build

ARG TARGETOS TARGETARCH
ARG POSTGRES_VERSION=14

RUN mkdir -p /workspace/build/src && \
    curl -sL https://github.com/pgvector/pgvector/archive/refs/tags/v0.4.2.tar.gz -o /workspace/pgvector_0.4.2.orig.tar.gz && \
    tar xf /workspace/pgvector_0.4.2.orig.tar.gz -C /workspace/build/src --strip-components 1

COPY build/debian /workspace/build/debian
COPY build/Makefile /workspace/build/Makefile
COPY build/buildkit /workspace/build/buildkit

WORKDIR /workspace/build

ENV DESTDIR /pg_ext
ENV POSTGRES_VERSION ${POSTGRES_VERSION}

RUN pg_buildext updatecontrol
RUN debuild \
    --prepend-path /usr/local/bin \
    --preserve-envvar CONF_EXTRA_VERSION \
    --preserve-envvar UNENCRYPTED_PACKAGE \
    --preserve-envvar PACKAGE_ENCRYPTION_KEY \
    --preserve-envvar MSRUSTUP_PAT \
    --preserve-envvar MSCODEHUB_USERNAME \
    --preserve-envvar MSCODEHUB_PASSWORD \
    -uc -us -B --lintian-opts --profile debian --allow-root

FROM scratch AS export

COPY --from=build /workspace/*.deb  .
