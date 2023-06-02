# Buildkit

A `pgxm` buildkit is a collection of files that provides instructions on how to build a PostgreSQL extension.
It includes a `buildkit.yaml` configuration file and a `build` script.

## buildkit.yaml

The `buildkit.yaml` is a YAML file that encapsulates configurations describing a buildkit. It contains the following fields:

```yaml
apiVersion: Specifies the buildkit API version. (required)
name: Gives the name of the buildkit. This name is usually the same as the extension name. (required)
version: Provides the buildkit's version, following the SemVer 2 versioning scheme. (required)
extVersion: Specifies the version of the extension that this buildkit is designed to build. This does not need to follow SemVer. Quotes are recommended to prevent interpretation as a number. (required)
pgVersion: Lists the PostgreSQL versions that this extension supports. (required)
dependencies: Lists the apt package dependencies at runtime. (optional)
buildDependencies: Lists the apt pacakge dependencies at build time. (optional)
description: Offers a brief, single-sentence description of this project. (optional)
homepage: Provides the URL to the project homepage. (optinal)
keywords: Contains a list of keywords to improve the extension's discoverability when listed in `pgxm search`. This is optional but can be useful for users searching for extensions. (optional)
maintainers: # Lists the maintainers of the project (optional)
  - name: The maintainers name (required for each maintainer)
    email: The maintainers email (optional for each maintainer)
    url: A URL for the maintainer (optional for each maintainer)
```

## build

The `build` script performs the actual build of the extension at the build stage.
This script is executed with the following environment variables set:

```dosini
PG_CONFIG=/usr/lib/postgresql/$pgVersion/bin/pg_config
USE_PGXS=1
```

This script is optional.

## install

The `install` script performs the actual install of the extension at the install stage.
This script is executed with the following environment variables set:

```dosini
DESTDIR=/workspace/build/debian/postgresql-$pgVersion-pgxm-$name
PG_CONFIG=/usr/lib/postgresql/$pgVersion/bin/pg_config
USE_PGXS=1
```

This script is required. The extension must be installed to `$DESTDIR`.

## clean

The `clean` script performs the actual clean of the extension at the clean stage.
This script is executed with the following environment variables set:

```dosini
PG_CONFIG=/usr/lib/postgresql/$pgVersion/bin/pg_config
USE_PGXS=1
```

This script is optional.
