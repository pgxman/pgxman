# syntax=docker/dockerfile:1

FROM ubuntu:22.04 as base

ARG POSTGRES_BASE_VERSION=14

RUN set -eux; \
  apt-get update; \
  export DEBIAN_FRONTEND=noninteractive; \
  apt-get install -y \
  gnupg \
  postgresql-common \
  git \
  ; \
  sh /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y; \
  apt-get update; \
  apt-get upgrade -y; \
  apt-get install -y  \
  lsb-release \
  gcc \
  make \
  libssl-dev \
  autoconf \
  pkg-config \
  postgresql-${POSTGRES_BASE_VERSION} \
  postgresql-server-dev-${POSTGRES_BASE_VERSION} \
  ; \
  rm -rf /var/lib/apt/lists/*

FROM base as builder

ARG TARGETOS TARGETARCH
ARG POSTGRES_BASE_VERSION=14

COPY build /workspace/build
WORKDIR /workspace

ENV DESTDIR /pg_ext
RUN /workspace/build ${POSTGRES_BASE_VERSION}
RUN tar -zcvf /pg_ext_${TARGETOS}_${TARGETARCH}.tar.gz /pg_ext

FROM scratch AS export

ARG TARGETOS TARGETARCH

COPY --from=builder /pg_ext_${TARGETOS}_${TARGETARCH}.tar.gz  .
