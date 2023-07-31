# PGXMan - PostgreSQL Extension Manager

Welcome to PGXMan, your one-stop solution for managing PostgreSQL extensions!

PostgreSQL extensions enhance the database's capabilities by introducing new features, data types, functions, and performance optimizations without altering the core code. PGXMan streamlines dealing with these extensions by simplifying the tasks of building, packaging, and installing them.

## Installation

### Mac

MacOS users can install `pgxman` via Homebrew:

```console
brew install pgxman/tap/pgxman
```

### Standalone

For a standalone installation, download the latest [compiled binaries](https://github.com/pgxman/release/releases/) and add them to your system's executable path.

### From source

If you have the [Go compiler](https://go.dev/dl/) installed, you can build and install `pgxman` from source:

```console
GOPRIVATE=github.com/pgxman/pgxman go install github.com/pgxman/pgxman/cmd/pgxman@latest
```

## Installing a PostgreSQL extension

Installing an extension with `pgxman` is straightforward. To install the `pgvector` extension version `0.4.4` for PostgreSQL 15:

```console
pgxman install pgvector=0.4.4@15
```

Verify the installation using your PostgreSQL instance:

```psql
postgres=# \dx
                                            List of installed extensions
        Name        | Version |   Schema   |                              Description
--------------------+---------+------------+----------------------------------------------------------------
 vector             | 0.4.4   |   public   | vector data type and ivfflat access method
 ...
(9 rows)
```

*Note*: `pgxman install` currently supports only Linux systems with the APT package manager. Support for more package managers is coming soon.

## Building a PostgreSQL extension

As an example, here's how to build the [pgvector](https://github.com/pgvector/pgvector) extension with `pgxman`.

1. **Initialize an extension manifest file**:

```console
pgxman init # follow the instruction
```

This command generates a manifest file named after your extension (`pgvector.yaml` in this example).
The file serves as your blueprint for building the extension.
The full specification is available [here](spec/extension.yaml.md).
Feel free to adapt the [official build manifest file](https://github.com/pgxman/buildkit/blob/main/buildkit/pgvector.yaml) for your requirements.

2. **Build the extension**:

```console
pgxman build -f pgvector.yaml
```

This will package the extension files in the `out` directory:

```console
$ ls out
postgresql-14-pgxman-pgvector_0.4.4_amd64.deb
postgresql-14-pgxman-pgvector_0.4.4_arm64.deb
postgresql-15-pgxman-pgvector_0.4.4_amd64.deb
postgresql-15-pgxman-pgvector_0.4.4_arm64.deb
```

*Note*: `pgxman build` only supports packaging extensions into Debian packages.
Other formats will be supported in future releases.
