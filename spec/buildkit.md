# Buildkit

A `pgxm` buildkit is a collection of files that provides instructions on how to build a PostgreSQL extension.
It includes a `buildkit.yaml` configuration file and a `build` script.

## buildkit.yaml

The `buildkit.yaml` is a YAML file that encapsulates configurations describing a buildkit. It contains the following fields:

```yaml
apiVersion: Specifies the buildkit API version. (required)
name: Gives the name of the buildkit. (required)
version: Provides the buildkit's version, following the SemVer 2 versioning scheme. (required)
extVersion: Specifies the version of the extension that this buildkit is designed to build. This does not need to follow SemVer. Quotes are recommended to prevent interpretation as a number. (required)
pgVersion: Lists the PostgreSQL versions that this extension supports. (required)
description: Offers a brief, single-sentence description of this project. (optional)
homepage: Provides the URL to the project homepage. (optinal)
keywords: Contains a list of keywords to improve the extension's discoverability when listed in `pgxm search`. This is optional but can be useful for users searching for extensions. (optional)
maintainers: # Lists the maintainers of the project (optional)
  - name: The maintainers name (required for each maintainer)
    email: The maintainers email (optional for each maintainer)
    url: A URL for the maintainer (optional for each maintainer)
```

## build

The `build` script performs the actual build of the extension.
This script is executed with the following environment variables set:

* `POSTGRES_VERSION`: The PostgreSQL version that the extension is being built.
* `DESTDIR`: The destination directory for the built extension. Your script should ensure that the built extension is output to this directory.

When the command `pgxm buildkit build` is executed, the `build` script is run to build the extension for each PostgreSQL version listed under `pgVersion` in the `buildkit.yaml` file.
