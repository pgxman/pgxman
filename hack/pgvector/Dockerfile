# syntax=docker/dockerfile:1

FROM ubuntu:22.04 as base

ARG POSTGRES_VERSION=14

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
  postgresql-${POSTGRES_VERSION} \
  postgresql-server-dev-${POSTGRES_VERSION} \
  ; \
  rm -rf /var/lib/apt/lists/*

FROM base as build

ARG TARGETOS TARGETARCH
ARG POSTGRES_VERSION=14

COPY build /workspace/build
WORKDIR /workspace

ENV DESTDIR /pg_ext
ENV POSTGRES_VERSION ${POSTGRES_VERSION}
RUN /workspace/build
RUN tar -zcvf /pgvector_${TARGETOS}_${TARGETARCH}.tar.gz /pg_ext

FROM scratch AS export

ARG TARGETOS TARGETARCH

COPY --from=build /pgvector_${TARGETOS}_${TARGETARCH}.tar.gz  .
