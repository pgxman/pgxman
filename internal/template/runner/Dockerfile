FROM {{ .RunnerImage }}

ARG PGXMAN_PACK_INSTALL_ARGS=""

COPY pgxman.yaml.tmp /pgxman/pgxman.yaml
COPY files /pgxman/files
RUN pgxman pack install --file /pgxman/pgxman.yaml --yes $PGXMAN_PACK_INSTALL_ARGS
