# syntax=docker/dockerfile:1

ARG BUILD_IMAGE
FROM $BUILD_IMAGE as build

ARG BUILD_SHA
ARG PGXMAN_PACK_ARGS=""

COPY extension.yaml /workspace/extension.yaml
WORKDIR /workspace

RUN pgxman-pack $PGXMAN_PACK_ARGS

FROM scratch AS export

COPY --from=build /workspace/target/*.deb  .
