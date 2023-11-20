FROM {{ .RunnerImage }}

ARG PGXMAN_BUNDLE_ARGS=""

COPY pgxman.yaml /pgxman/pgxman.yaml
COPY files /pgxman/files
RUN pgxman bundle --file /pgxman/pgxman.yaml --yes $PGXMAN_BUNDLE_ARGS
