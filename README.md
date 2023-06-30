# PGXMan - PostgreSQL Extension Manager

Welcome to PGXMan, your one-stop solution for managing PostgreSQL extensions!

In PostgreSQL, extensions amplify the database's capabilities by introducing new features, data types, functions, and performance optimizations without modifying the core code. PGXMan is designed to streamline the process of dealing with these extensions by simplifying the tasks of building, packaging, and installing them.

## Installation

### Mac

For MacOS users, `pgxman` can be installed via Homebrew:

```console
brew install pgxman/tap/pgxman
```

### Standalone

`pgxman` can be installed as a standalone executable. Download the latest [compiled binaries](https://github.com/pgxman/release/releases/) and add it to your system's executable path.

### From source

If you have the [Go compiler](https://go.dev/dl/) installed, `pgxman` can be installed from source:

```console
GOPRIVATE=github.com/pgxman/pgxman go install github.com/pgxman/pgxman/cmd/pgxman@latest
```

## Installing a PostgreSQL extension

With pgxman, installing an extension is as simple as specifying the extension name and version.
To install `pgvector` version `0.4.4` for PostgreSQL 15, for example, you would run:

```console
pgxman install pgvector=0.4.4@15
```

This command will install the `pgvector` extension onto your local PostgreSQL instance. To verify a successful installation, you can inspect your PostgreSQL instance:

```psql
postgres=# \dx
                                            List of installed extensions
        Name        | Version |   Schema   |                              Description
--------------------+---------+------------+----------------------------------------------------------------
 vector             | 0.4.4   | public     | vector data type and ivfflat access method
 ...
(9 rows)
```

Please note `pgxman install` currently only supports Linux systems with the APT package manager.
We plan to extend this in future releases to support more package managers.

## Building a PostgreSQL extension

Let's walk through an example of how to build the [pgvector](https://github.com/pgvector/pgvector) extension.

Start by creating a directory named `pgvector` and initializing it using `pgxman init`:

```console
mkdir pgvector
cd pgvector
pgxman init
```

This command generates an `extension.yaml` file:

```console
$ tree pgvector
pgvector
└── extension.yaml

1 directory, 1 files
```

The `extension.yaml` file serves as your blueprint for building the extension.
You can find its full specification [here](spec/extension.yaml.md)
Feel free to adapt the example from [examples/pgvector](examples/pgvector) to meet your needs.

Once your `extension.yaml` is all set, initiate the build process:

```console
pgxman build
```

Successful completion of the build process will result in the built extension files, neatly packaged in the `out` directory within the `pgvector` folder:

```console
$ out
├── postgresql-14-pgxman-pgvector_0.4.4_amd64.deb
├── postgresql-14-pgxman-pgvector_0.4.4_arm64.deb
├── postgresql-15-pgxman-pgvector_0.4.4_amd64.deb
└── postgresql-15-pgxman-pgvector_0.4.4_arm64.deb

1 directory, 4 files
```

Currently, `pgxman build` only supports packaging extensions into Debian packages.
In future releases, we plan to extend this support to other packaging formats like RPM, APK, etc..
