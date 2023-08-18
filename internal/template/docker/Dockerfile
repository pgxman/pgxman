# syntax=docker/dockerfile:1

ARG BUILD_IMAGE
FROM $BUILD_IMAGE as build

ARG BUILD_SHA
ARG PGXMAN_PACK_ARGS=""

COPY extension.yaml /workspace/extension.yaml
WORKDIR /workspace

RUN pgxman-pack init $PGXMAN_PACK_ARGS
RUN pgxman-pack pre $PGXMAN_PACK_ARGS
RUN pgxman-pack main $PGXMAN_PACK_ARGS
RUN pgxman-pack post $PGXMAN_PACK_ARGS
