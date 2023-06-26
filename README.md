# PGXMan - PostgreSQL Extension Manager

Welcome to PGXMan, the ultimate PostgreSQL Extension Manager!

In PostgreSQL, extensions play a pivotal role by introducing new features, data types, functions, or performance optimizations. These modules amplify PostgreSQL's power without changing its core code. Whether you need to meet unique requirements, enhance performance, or integrate with different systems, extensions are your key. PGXMan is here to make the process of dealing with extensions a breeze by streamlining the tasks of building, packaging, and installing them.

## Building a PostgreSQL extension

As an example, let's demonstrate how to build an extension using [pgvector](https://github.com/pgvector/pgvector).

Start by creating a directory named `pgvector` and initialize it using `pgxman init`:

```console
mkdir pgvector
cd pgvector
pgxman init
```

This command yields an `extension.yaml` file:

```console
$ tree pgvector
pgvector
└── extension.yaml

1 directory, 1 files
```

The `extension.yaml` file is your blueprint for building the extension. The full specification of the file can be found [here](spec/extension.yaml.md).
Feel free to adapt the example from [examples/pgvector](examples/pgvector) according to your needs.

Once your extension.yaml is all set, kick-start the build process:

```console
pgxm build
```

Successful execution will package the built extension files neatly in the `out` directory of the `pgvector` folder:

```console
$ out
├── postgresql-14-pgxman-pgvector_0.4.2_amd64.deb
├── postgresql-14-pgxman-pgvector_0.4.2_arm64.deb
├── postgresql-15-pgxman-pgvector_0.4.2_amd64.deb
└── postgresql-15-pgxman-pgvector_0.4.2_arm64.deb

1 directory, 4 files
```

To make your extension available to the wider PostgreSQL community, publish it to the `pgxman` hub:

```console
pgxman publish
```

## Installing a PostgreSQL extension

Installing an extension with `pgxman` is as straightforward as specifying the extension name and version. For instance, to install version `pgvector` version `0.4.2`:

```console
pgxman install pgvector=0.4.2@14
```

This command installs the `pgvector` extension onto your local PostgreSQL instance. To verify a successful installation, inspect your PostgreSQL instance:

```psql
postgres=# \dx
                                            List of installed extensions
        Name        | Version |   Schema   |                              Description
--------------------+---------+------------+----------------------------------------------------------------
 vector             | 0.4.2   | public     | vector data type and ivfflat access method
 ...
(9 rows)
```

Here, you'll see the newly installed `pgvector` extension and other pre-existing extensions.
