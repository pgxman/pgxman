# syntax=docker/dockerfile:1

ARG BUILD_IMAGE=notexist
FROM $BUILD_IMAGE AS build

ARG BUILD_SHA
ARG PARALLEL=""
ARG PGXMAN_PACK_ARGS=""
ARG WORKSPACE_DIR

RUN mkdir ${WORKSPACE_DIR}
WORKDIR ${WORKSPACE_DIR}
COPY extension.yaml ${WORKSPACE_DIR}/extension.yaml

RUN pgxman-pack init $PGXMAN_PACK_ARGS
RUN pgxman-pack pre $PGXMAN_PACK_ARGS
RUN pgxman-pack main --parallel $PARALLEL $PGXMAN_PACK_ARGS
RUN pgxman-pack post $PGXMAN_PACK_ARGS
