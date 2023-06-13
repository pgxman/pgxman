# PGXMan - PostgreSQL Extension Manager

Welcome to PGXMan!
This robust tool is designed to streamline your PostgreSQL Extensions management, making the tasks of building, packaging, and installing them a breeze.
PGXman is a crucial ally for both extension developers and PostgreSQL users seeking to augment their database capabilities.
It effectively simplifies your workflow and reduces the intricacies of extension management.

## Building a PostgreSQL extension

The process of building an extension involves a few key steps.
Let's walk through an example where we're building the [pgvector](https://github.com/pgvector/pgvector) extension.

First, create a directory called `pgvector` and run `pgxman init`:

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

The `extension.yaml` file is a configuration file that outlines how your extension should be built. The spec of the file is available [here](spec/extension.yaml.md).
You can take inspiration from the example found in [examples/pgvector](examples/pgvector) and adjust it to fit your needs.

Once the file is ready, you can build the extension with:

```console
pgxm buildkit build
```

Upon a successful build, the extension files will be packaged in the `out` directory under the `pgvector` folder:

```console
$ out
├── postgresql-14-pgxman-pgvector_0.4.2_amd64.deb
├── postgresql-14-pgxman-pgvector_0.4.2_arm64.deb
├── postgresql-15-pgxman-pgvector_0.4.2_amd64.deb
└── postgresql-15-pgxman-pgvector_0.4.2_arm64.deb

1 directory, 4 files
```

To make the built extension available for others to use, publish it to the `pgxman` hub:

```console
pgxman publish
```

## Installing a PostgreSQL extension

With `pgxman`, installing an extension is straightforward. You just need to specify the extension name and the version number. For example, to install version 0.4.2 of `pgvector`, use this command:

```console
pgxman install pgvector@0.4.2
```

The above command will install the `pgvector` extension to your local PostgreSQL instance. To validate the successful installation, you can inspect your PostgreSQL instance for the newly installed extension:

```psql
postgres=# \dx
                                            List of installed extensions
        Name        | Version |   Schema   |                              Description
--------------------+---------+------------+----------------------------------------------------------------
 vector             | 0.4.2   | public     | vector data type and ivfflat access method
 ...
(9 rows)
```
