# PGXMan - PostgreSQL Extension Manager

Welcome to PGXMan, the solution for managing PostgreSQL extensions!

PostgreSQL extensions enhance the database's capabilities by introducing new
features, data types, functions, and performance optimizations without altering
the core code. PGXMan streamlines using these extensions by simplifying the
tasks of building, packaging, and installing them.

PGXMan is currently compatible with the following Linux distributions:

- [Debian Bookworm](https://www.debian.org/releases/bookworm)
- [Ubuntu Jammy](https://releases.ubuntu.com/jammy)

For more information about pgxman, see our [full documentation](docs/README.md).

## Installation

Run:

```console
curl -sfL https://github.com/pgxman/release/releases/latest/download/install.sh | sh -
```

Or download the latest [compiled binaries](https://github.com/pgxman/release/releases/) and add them to your executable path.

For further detailed instructions, see [our documentation](docs/installing_pgxman.md).

## Quickstart

### `install`

To install an extension, such as the `pgvector` version `0.5.0` for PostgreSQL 15, run:

```console
pgxman install pgvector=0.5.0@15
```

You can also utilize a [pgxman.yaml](spec/pgxman.yaml.md) file to install multiple extensions at once:

```console
pgxman install -f pgxman.yaml
```

Once installed, restart Postgres, then `CREATE EXTENSION pgvector`.

### `search`

Find extensions with `pgxman search`:

```console
pgxman search fdw
```

### `init`, `build`

[Please refer to our docs for how to build an extension for pgxman](docs/building_an_extension.md).

## How it works

pgxman's build system works with your system package manager. The buildkit
specifies how to build each extension and builds it for each package manager.
When a buildkit is added or updated, a build is conducted and the packages are
stored in pgxman's repositories.

When installing an extension, pgxman's package repository is used to install
the packages into your system. This way, pgxman is able to handle dependency
management, installation, and uninstallation through your system's package
manager.

For more details, see [How It Works](docs/how_it_works.md) in the
documentation.

## Roadmap

- Support for multiple operating systems & package managers
  - [x] Debian Bookworm & APT
  - [x] Ubuntu Jammy & APT
  - [ ] MacOS & Homebrew
  - [ ] Fedora & RPM
  - [ ] AlmaLinux & RPM

- Support popular extensions
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

- CLI
  - [x] `pgxman init` to create a buildkit YAML file by following a questionnaire
  - [x] `pgxman build` to build & package an extension
  - [x] `pgxman search` to search for an extension
  - [x] `pgxman install` to install an extension
  - [ ] `pgxman upgrade` to upgrade an extension
  - [ ] `pgxman uninstall` to uninstall an extension
  - [ ] `pgxman container` to explore extensions in a container

- [ ] Website
- [ ] API
