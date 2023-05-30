# pgxm - PostgreSQL Extension Manager

Welcome to `pgxm`! This is a comprehensive tool that simplifies the management of PostgreSQL extensions, making it easier than ever to build, package, and install them.
Whether you're an extension developer or a PostgreSQL user looking to expand your database capabilities, `pgxm` can streamline your workflow and minimize the complexity of extension management.

## Building a PostgreSQL extension

The process of building an extension involves a few key steps. Let's walk through an example where we're building the [pgvector](https://github.com/pgvector/pgvector) extension.

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

Refer to the example provided in the [examples/pgvector](examples/pgvector) directory of this repository, and adjust your build and buildkit.yaml files accordingly.

Once these files are set up, you're ready to build the extension:

```console
pgxm buildkit build
```

Upon successful execution of the build command, the built and packaged extension files are placed in the out directory under `pgvector`:

```console
pgvector/out
├── linux_amd64
│   └── pgvector_linux_amd64.tar.gz
└── linux_arm64
    └── pgvector_linux_arm64.tar.gz
```

To build & share the extension with others, publish it to the `pgxm` hub:

```console
pgxm publish
```

## Installing a PostgreSQL extension

With `pgxm`, installing an extension is straightforward. You just need to specify the extension name and the version number. For example, to install version 0.4.2 of `pgvector`, use this command:

```console
pgxm install pgvector@0.4.2
```

This command installs pgvector to your local PostgreSQL instance. To confirm the installation, you can check your PostgreSQL instance for the new extension:

```psql
postgres=# \dx
                                            List of installed extensions
        Name        | Version |   Schema   |                              Description
--------------------+---------+------------+------------------------------------------------------------------------
 columnar           | 11.1-6  | public     | Hydra Columnar extension
 file_fdw           | 1.0     | public     | foreign-data wrapper for flat file access
 pg_auth_mon        | 1.1     | public     | monitor connection attempts per user
 pg_stat_kcache     | 2.2.1   | public     | Kernel statistics gathering
 pg_stat_statements | 1.9     | public     | track planning and execution statistics of all SQL statements executed
 plpgsql            | 1.0     | pg_catalog | PL/pgSQL procedural language
 plpython3u         | 1.0     | pg_catalog | PL/Python3U untrusted procedural language
 set_user           | 3.0     | public     | similar to SET ROLE but with added logging
 vector             | 0.4.2   | public     | vector data type and ivfflat access method
(9 rows)
```
