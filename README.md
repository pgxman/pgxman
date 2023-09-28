# PGXMan - PostgreSQL Extension Manager

Welcome to PGXMan, your one-stop solution for managing PostgreSQL extensions!

PostgreSQL extensions enhance the database's capabilities by introducing new features, data types, functions, and performance optimizations without altering the core code. PGXMan streamlines dealing with these extensions by simplifying the tasks of building, packaging, and installing them.

## Installation

PGXMan is compatible with the following Debian-based Linux distributions:

- [Debian Bookworm](https://www.debian.org/releases/bookworm)
- [Ubuntu Jammy](https://releases.ubuntu.com/jammy)

### Prerequisites

- Docker
- Apt Package Manager
- PostgreSQL

### Installer

The simplest method to install `pgxman` is through the installer script:

```console
curl -sfL https://github.com/pgxman/release/releases/latest/download/install.sh | sh -
```

### Manual

Alternatively, download the latest [compiled Linux binaries](https://github.com/pgxman/release/releases/) and add them to your system's executable path.

### From source

If you have the [Go compiler](https://go.dev/dl/) installed, you can build and install `pgxman` from source:

```console
GOPRIVATE=github.com/pgxman/pgxman go install github.com/pgxman/pgxman/cmd/pgxman@latest
```

## Installing a PostgreSQL extension

### Individual Extension

Installing an extension, such as the `pgvector` version `0.4.4` for PostgreSQL 15, is as simple as running:

```console
pgxman install pgvector=0.4.4@15
```

### Batch Installation

You can also utilize a [pgxman.yaml](spec/pgxman.yaml.md) file to install multiple extensions at once:

```console
cat <<EOS > pgxman.yaml
apiVersion: v1
extensions:
  - name: "pgvector"
    version: "0.4.4"
  - name: "pg_ivm"
    version: "1.5.1"
pgVersions:
  - "15"
EOS
pgxman install -f pgxman.yaml
```

### Verification

To verify the successful installation of extensions, execute the following command on your PostgreSQL instance:

```psql
postgres=# \dx
                                            List of installed extensions
        Name        | Version |   Schema   |                              Description
--------------------+---------+------------+----------------------------------------------------------------
 vector             | 0.4.4   |   public   | vector data type and ivfflat access method
 ...
(9 rows)
```

## Building a PostgreSQL extension

As an example, here's how to build the [pgvector](https://github.com/pgvector/pgvector) extension with `pgxman`.

1. **Initialize an extension manifest file**:

```console
pgxman init # follow the instruction
```

This command generates a manifest file named `extension.yaml`.
The file serves as your blueprint for building the extension.
[Please refer to the full buildkit file specification for details.](https://github.com/pgxman/buildkit/blob/main/spec/buildkit.md).

2. **Build the extension**:

```console
pgxman build -f extension.yaml
```

This will package the extension files in the `out` directory:

```console
$ tree out
out
├── linux_amd64
│   ├── debian
│   │   └── bookworm
│   │       ├── postgresql-13-pgxman-pgvector_0.4.4_amd64.deb
│   │       ├── postgresql-14-pgxman-pgvector_0.4.4_amd64.deb
│   │       └── postgresql-15-pgxman-pgvector_0.4.4_amd64.deb
│   └── ubuntu
│       └── jammy
│           ├── postgresql-13-pgxman-pgvector_0.4.4_amd64.deb
│           ├── postgresql-14-pgxman-pgvector_0.4.4_amd64.deb
│           └── postgresql-15-pgxman-pgvector_0.4.4_amd64.deb
└── linux_arm64
    ├── debian
    │   └── bookworm
    │       ├── postgresql-13-pgxman-pgvector_0.4.4_arm64.deb
    │       ├── postgresql-14-pgxman-pgvector_0.4.4_arm64.deb
    │       └── postgresql-15-pgxman-pgvector_0.4.4_arm64.deb
    └── ubuntu
        └── jammy
            ├── postgresql-13-pgxman-pgvector_0.4.4_arm64.deb
            ├── postgresql-14-pgxman-pgvector_0.4.4_arm64.deb
            └── postgresql-15-pgxman-pgvector_0.4.4_arm64.deb

11 directories, 12 files
```

## Roadmap

- [ ] Support for multiple Linux distros & package managers
  - [x] Debian Bookworm & APT
  - [x] Ubuntu Jammy & APT
  - [ ] MacOS & Homebrew
  - [ ] Fedora & RPM
  - [ ] AlmaLinux & RPM

- [ ] Support popular extensions
  - [x] pgvector
  - [x] pg_ivm
  - [x] mysql_fdw
  - [x] parquet_s3_fdw
  - [x] hydra_columnar
  - [x] multicorn2
  - [x] pg_hint_plan
  - [x] psql-http
  - [ ] postgis
  - [ ] hll
  - [ ] pg-embedding
  - [ ] plv8
  - [ ] pg_graphql

- [ ] CLI
  - [x] `pgxman init` to create a buildkit YAML file by following a questionnaire
  - [x] `pgxman build` to build & package an extension
  - [x] `pgxman search` to search for an extension
  - [x] `pgxman install` to install an extension
  - [ ] `pgxman upgrade` to upgrade an extension
  - [ ] `pgxman uninstall` to uninstall an extension
  - [ ] `pgxman container` to explore extensions in a container
