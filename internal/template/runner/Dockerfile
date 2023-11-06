FROM {{ .RunnerImage }}

COPY pgxman.yaml /tmp/pgxman.yaml
RUN pgxman bundle -f /tmp/pgxman.yaml
