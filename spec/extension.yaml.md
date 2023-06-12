# `extension.yaml` Specification

The `extension.yaml` file is a configuration file in YAML format, used by `pgxm` to specify how a PostgreSQL extension should be built and installed.

## `apiVersion`

- **Description**: Specifies the API version that the buildkit adheres to. This is to ensure compatibility with the buildkit.
- **Type**: String
- **Required**: Yes
- **Supported Values**: Currently, only `v1` is supported.

## `name`

- **Description**: The name of the extension.
- **Type**: String
- **Required**: Yes

## `maintainers`

- **Description**: Contains information about the individuals or organizations responsible for the maintenance of the extension.
- **Type**: List of objects
- **Required**: Yes
- **Object Fields**:
  - `name`: The full name of the maintainer. (String, Required)
  - `email`: Email address to contact the maintainer. (String, Required)

## `source`

- **Description**: The URL where the extension's source code can be downloaded. The source code must be in `tar.gz` format.
- **Type**: String
- **Required**: Yes

## `version`

- **Description**: Specifies the version of the extension. It is not required to follow Semantic Versioning, but it is recommended to use quotes to prevent the YAML parser from interpreting it as a number.
- **Type**: String
- **Required**: Yes

## `pgVersions`

- **Description**: A list specifying the versions of PostgreSQL that the extension is compatible with. If this field is not specified, it is assumed that the extension is compatible with all versions.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `"13"`, `"14"`, `"15"`
- **Default Values**: `"13"`, `"14"`, `"15"`

## `build`

- **Description**: A Bash script that automates the build process of the extension. Certain environment variables are set during the execution of this script. The built extension must be placed in the directory specified by `$DESTDIR`.
- **Type**: String (Bash script)
- **Required**: Yes
- **Environment Variables**:
  - `DESTDIR`: Specifies the directory where the built extension should be placed.
  - `PG_CONFIG`: Specifies the path to the `pg_config` executable.
  - `USE_PGXS`: Always set to `1`.

The following is an example:

```yaml
build: |
  make
  DESTDIR=${DESTDIR} make install
```

## `dependencies`

- **Description**: Specifies a list of packages that must be present for the extension to run.
- **Type**: List of strings
- **Required**: No

## `buildDependencies`

- **Description**: Specifies a list of packages that must be present for building the extension.
- **Type**: List of strings
- **Required**: No

## `arch`

- **Description**: Specifies the architectures that the extension supports. If not specified, it is assumed that the extension supports all architectures.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `amd64`, `arm64`
- **Default Values**: `amd64`, `arm64`

## `platform`

- **Description**: Specifies the platforms that the extension supports. Currently, only Linux is supported.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `linux`
- **Default Values**: `linux`

## `formats`

- **Description**: Specifies the formats in which the built extension can be exported. Currently, only Debian packages (`deb`) are supported.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `deb`
- **Default Values**: `deb`

## `description`

- **Description**: Provides a brief and concise description of the extension.
- **Type**: String
- **Required**: No

## `homepage`

- **Description**: Specifies the URL of the extension's homepage, where users can find additional information about the extension.
- **Type**: String
- **Required**: No

## `keywords`

- **Description**: Specifies a list of keywords relevant to the extension. These keywords can enhance the extension's discoverability through searches within the `pgxm` tool.
- **Type**: List of strings
- **Required**: No
