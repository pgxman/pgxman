# pgxm - PostgreSQL Extension Manager

Welcome to `pgxm`!
This robust tool is designed to streamline your PostgreSQL extensions management, making the tasks of building, packaging, and installing them a breeze.
`pgxm` is a crucial ally for both extension developers and PostgreSQL users seeking to augment their database capabilities.
It effectively simplifies your workflow and reduces the intricacies of extension management.

## Building a PostgreSQL extension

The process of building an extension involves a few key steps.
Let's walk through an example where we're building the [pgvector](https://github.com/pgvector/pgvector) extension.

First, create a new `pgxm` buildkit:

```console
pgxm buildkit new pgvector
```

This command generates a new folder named `pgvector` in your current directory, with the following structure:

```console
$ tree pgvector
pgvector
├── build
└── buildkit.yaml

1 directory, 2 files
```

* `buildkit.yaml` is a configuration file that outlines how your extension should be built.
* `build` is a script that, when run, builds the extension.

The spec of a buildkit is avaialbe [here](spec/buildkit.md).
You can take inspiration from the example found in [examples/pgvector](examples/pgvector) and adjust your `build` and `buildkit.yaml` files to fit your needs.

Once these files are set up, you're ready to build the extension:

```console
pgxm buildkit build
```

Upon a successful build, the extension files will be packaged in the `out` directory under the `pgvector` folder:

```console
pgvector/out
├── linux_amd64
│   └── pgvector_linux_amd64.tar.gz
└── linux_arm64
    └── pgvector_linux_arm64.tar.gz
```

To make the built extension available for others to use, publish it to the `pgxm` hub:

```console
pgxm publish
```

## Installing a PostgreSQL extension

With `pgxm`, installing an extension is straightforward. You just need to specify the extension name and the version number. For example, to install version 0.4.2 of `pgvector`, use this command:

```console
pgxm install pgvector@0.4.2
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
