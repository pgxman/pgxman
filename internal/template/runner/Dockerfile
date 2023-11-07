FROM {{ .RunnerImage }}

COPY pgxman.yaml /pgxman/pgxman.yaml
COPY files /pgxman/files
RUN pgxman bundle -f /pgxman/pgxman.yaml
