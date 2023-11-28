# pgxman - PostgreSQL Extension Manager

[![GitHub release](https://img.shields.io/github/release/pgxman/pgxman.svg)](https://github.com/pgxman/pgxman/releases)

Welcome to pgxman, the solution for managing PostgreSQL extensions!

PostgreSQL extensions enhance the database's capabilities by introducing new
features, data types, functions, and performance optimizations without altering
the core code. **pgxman** streamlines using these extensions by simplifying the
tasks of building, packaging, and installing them.

## More Documentation

Try `pgxman help`, `man pgxman`, or [read our documentation](https://docs.pgxman.com).

## Installation

Run:

```sh
# with homebrew
brew install pgxman/tap/pgxman
# without homebrew
curl -sfL https://install.pgx.sh | sh -
```

For more options, see [our installation documentation](https://docs.pgxman.com/installing_pgxman).

## Quickstart

### `search`

Find extensions with `pgxman search`:

```sh
pgxman search fdw
```

### `install`

To install an extension, say `pgvector`, run:

```sh
pgxman install pgvector
```

pgxman will automatically detect your local install of Postgres (or, on MacOS, will [use a container](https://docs.pgxman.com/container)).

You can specify multiple extensions, specific extension versions, and a PG version:

```sh
pgxman install --pg 15 pgvector=0.5.1 pg_ivm=1.7.0
```

You can also utilize a [pack file](https://docs.pgxman.com/spec/pack) to install multiple extensions at once:

```sh
pgxman pack install # installs from pgxman.yaml from current directory
pgxman pack install -f /path/to/pgxman.yaml
```

Once installed, restart Postgres, then use `CREATE EXTENSION`.

### `init`, `build`

[Please refer to our docs for how to build an extension for pgxman](https://docs.pgxman.com/building_an_extension).

## How it works

pgxman's build system works with your system package manager. The buildkit
specifies how to build each extension and builds it for each package manager.
When a buildkit is added or updated, a build is conducted and the packages are
stored in pgxman's repositories.

When installing an extension, pgxman's package repository is used to install
the packages into your system. This way, pgxman is able to handle dependency
management, installation, and uninstallation through your system's package
manager.

pgxman itself is either installed as an apt package or via homebrew.

For more details, see [how it works](https://docs.pgxman.com/how_it_works) in the
documentation.

## License

The pgxman client is licensed under the [FSL](LICENSE.md), which, in short, means
pgxman is open for all internal, non-competing usage. To learn more about the
FSL, please see [fsl.software](https://fsl.software). As stated:

> You can do anything with FSL software except undermine its producer. You can read it,
> learn from it, run it internally, modify it, and propose improvements back to the
> producer. After two years it becomes Open Source software under Apache 2.0 or MIT.

We consider any Postgres service provider using pgxman as part of their service
to be a Competing Usage. However, we encourage widespread adoption of pgxman and welcome
any service provider to contact us at `pgxman [at] hydra [dot] so` to obtain a
license for usage as part of your service. Our main concern is assuring the pgxman
service can scale to the needs of your service.
