# syntax=docker/dockerfile:1

ARG BUILD_SHA
ARG BUILD_IMAGE

FROM $BUILD_IMAGE as build

COPY extension.yaml /workspace/extension.yaml
WORKDIR /workspace

RUN pgxman-pack

FROM scratch AS export

COPY --from=build /workspace/target/*.deb  .
