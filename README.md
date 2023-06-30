# PGXMan - PostgreSQL Extension Manager

Welcome to PGXMan, your one-stop solution for managing PostgreSQL extensions!

In PostgreSQL, extensions amplify the database's capabilities by introducing new features, data types, functions, and performance optimizations, all without modifying the core code. They're your go-to when it comes to tailoring PostgreSQL to meet unique requirements, enhancing performance, or integrating with various systems. PGXMan is designed to streamline the process of dealing with these extensions by simplifying the tasks of building, packaging, and installing them.

## Building a PostgreSQL extension

For instance, let's learn how to build the [pgvector](https://github.com/pgvector/pgvector) extension.

Create a directory named `pgvector` and initialize it using `pgxman init`:

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

The `extension.yaml` file is your blueprint for building the extension.
You can find its full specification [here](spec/extension.yaml.md)
Feel free to adapt the example from [examples/pgvector](examples/pgvector) to meet your needs.

Once your extension.yaml is all set, kick-start the build process:

```console
pgxman build
```

Successful completion of the build process will result in the newly built extension files, neatly packaged in the `out` directory within the `pgvector` folder:

```console
$ out
├── postgresql-14-pgxman-pgvector_0.4.4_amd64.deb
├── postgresql-14-pgxman-pgvector_0.4.4_arm64.deb
├── postgresql-15-pgxman-pgvector_0.4.4_amd64.deb
└── postgresql-15-pgxman-pgvector_0.4.4_arm64.deb

1 directory, 4 files
```

Currently, `pgxman build` supports packaging extensions into Debian packages.
In future releases, we plan to extend this support to other packaging formats like RPM, APK, etc..

## Installing a PostgreSQL extension

Installing an extension with `pgxman` is as straightforward as specifying the extension name and version.
For instance, to install version `pgvector` version `0.4.2` for PostgreSQL 14:

```console
pgxman install pgvector=0.4.4@14
```

This command will install the `pgvector` extension onto your local PostgreSQL instance. To confirm a successful installation, you can inspect your PostgreSQL instance:

```psql
postgres=# \dx
                                            List of installed extensions
        Name        | Version |   Schema   |                              Description
--------------------+---------+------------+----------------------------------------------------------------
 vector             | 0.4.4   | public     | vector data type and ivfflat access method
 ...
(9 rows)
```

Please note `pgxman install` currently supports Linux systems with the APT package manager only.
In future releases, we plan to extend this support more package managers.
