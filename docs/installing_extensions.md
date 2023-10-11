# Installing a PostgreSQL extension

## Individual Extension

Installing an extension, such as `pgvector` for PostgreSQL
15, is as simple as running:

```console
pgxman install pgvector@15
```

You can also specify the desired version of the extension by using `=version`:

```console
pgxman install pgvector=0.5.0@15
```

The Postgres version must always be specified by using `@`. Currently, Postgres
13-16 are supported.

As `pgxman` harnesses system's package manager for extension management,
admin privilleges might be required. To install extensions with `sudo`,
append the `--sudo` flag:

```console
pgxman install pgvector=0.5.0@15 --sudo
```

## Batch Installation using a pgxman file

You can also utilize a [pgxman.yaml](spec/pgxman.yaml.md) file to install
multiple extensions at once:

```console
$ cat <<EOF >pgxman.yaml
apiVersion: v1
extensions:
  - name: "pgvector"
    version: "0.5.0"
  - name: "pg_ivm"
    version: "1.5.1"
pgVersions:
  - "15"
EOF

$ pgxman install -f pgxman.yaml
```

## Verification

⚠️ Postgres needs to be restarted to see newly installed extensions.

To verify the successful installation of extensions, execute the following
command on your PostgreSQL instance:

```psql
postgres=# \dx
                                            List of installed extensions
        Name        | Version |   Schema   |                              Description
--------------------+---------+------------+----------------------------------------------------------------
 vector             | 0.5.0   |   public   | vector data type and ivfflat access method
 ...
(9 rows)
```

To install the extension into a database, use [`CREATE
EXTENSION`](https://www.postgresql.org/docs/current/sql-createextension.html).
