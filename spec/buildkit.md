# Buildkit Specification

PGXMan buildkit is a configuration file in YAML format that `pgxman` uses to specify how a PostgreSQL extension should be built and installed.

## `apiVersion`

- **Description**: Defines the API version to which the buildkit conforms, ensuring its compatibility.
- **Type**: String
- **Required**: Yes
- **Supported Values**: Currently, only `v1` is supported.

## `name`

- **Description**: Identifies the name of the extension.
- **Type**: String
- **Required**: Yes

## `maintainers`

- **Description**: Lists the individuals or organizations responsible for maintaining the extension.
- **Type**: List of objects
- **Required**: Yes
- **Object Fields**:
  - `name`: The maintainer's full name. (String, Required)
  - `email`: The maintainer's contact email. (String, Required)

## `source`

- **Description**: Specifies the URL for downloading the extension's source code, which must be in in `tar.gz` format.
- **Type**: String
- **Required**: Yes

## `version`

- **Description**: Specifies the version of the extension buildkit. The versioning must adhere to the [Semantic Versioning 2.0.0](https://semver.org) format.
- **Type**: String
- **Required**: Yes

## `pgVersions`

- **Description**: Lists the PostgreSQL versions compatible with the extension. If unspecified, the extension is assumed to be compatible with all versions.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `"13"`, `"14"`, `"15"`
- **Default Values**: `"13"`, `"14"`, `"15"`

## `build`

- **Description**: Contains Bash scripts that automates the extension's build process. Some environment variables are set during the execution of these scripts. The built extension must be placed in the `$DESTDIR` directory.
- **Type**: Object
- **Required**: Yes
- **Object fields**:
  - `pre`: A list of steps before the main build process. (List of objects, Optional)
    - `name`: Name of the build step. (String, Required)
    - `run`: Bash command for the build step. (String, Required)
  - `main`: A list of steps to be executed during main build process. (List of objects, Required)
    - `name`: Name of the build step. (String, Required)
    - `run`: Bash command for the build step. (String, Required)
    - **Environment Variables**:
      - `DESTDIR`: Indicates the directory where the built extension should be placed.
      - `WORKDIR`: The working directory that contains the source code and the script.
      - `PG_CONFIG`: Identifies the path to the `pg_config` executable.
      - `USE_PGXS`: Always set to `1`.
      - `PG_VERSION`: The PostgreSQL version that the script is building against.
  - `post`: A list of steps after the main build process. (List of objects, Optional)
    - `name`: Name of the build step. (String, Required)
    - `run`: Bash command for the build step. (String, Required)

The following is an example:

```yaml
build: |
  make
  DESTDIR=${DESTDIR} make install
```

## `dependencies`

- **Description**:  Lists the rpackages needed for the extension to function properly at runtime.
- **Type**: List of strings
- **Required**: No

## `buildDependencies`

- **Description**: Lists the packages necessary for building the extension.
- **Type**: List of strings
- **Required**: No

## `arch`

- **Description**: Lists the architectures that the extension supports. If unspecified, all architectures are assumed to be supported.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `amd64`, `arm64`
- **Default Values**: `amd64`, `arm64`

## `platform`

- **Description**: Lists the platforms that the extension supports. Currently, only Linux is supported.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `linux`
- **Default Values**: `linux`

## `formats`

- **Description**: Lists the formats in which the built extension can be packaged. Currently, only Debian packages (`deb`) are supported.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `deb`
- **Default Values**: `deb`

## `description`

- **Description**: Provides a succinct overview of the extension.
- **Type**: String
- **Required**: No

## `homepage`

- **Description**: Specifies the URL of the extension's homepage, where users can find additional information about the extension.
- **Type**: String
- **Required**: No

## `keywords`

- **Description**: Lists keywords relevant to the extension, aiding its discovery in `pgxman` tool searches.
- **Type**: List of strings
- **Required**: No

## `deb`

- **Description**: Configures settings specific to Debian.
- **Type**: Object
- **Required**: No
- **Object Fields**:
  - `buildDepdencies`: Additional list of Debian packages required for building the extension. (List of strings, Optional)
  - `Dependencies`: Additional list of Debian packages required for running the extension. (List of strings, Optional)
  - `AptRepositories`: Lists the APT repositories containing the above Debian packages. (List of objects, Optional)
    - `id`: The repository's id. (String, Required)
    - `types`: The repository's types. (String, Required, Supported Values: `deb`, `deb-src`)
    - `uris`: The repository's URIs. (List of strings, Required)
    - `suites`: The repository's suites within the APT repository. (List of objects, Required)
    - `components`: The components of the APT repository. (List of strings, Required)
    - `signedKey`: The GPG signed key of the APT repository. (List of objects, Required)
      - `uri`: The HTTP URL to download the GPG key. (String, Required)
      - `format`: The format of the GPG key. (String, Required, Supported Values: `gpg`, `asc`)
    - `target`: The apt repository's target platform. This can be a Linux codename (e.g., `bookworm`) or a Linux distribution (e.g., `debian`). If unspecified, the suite supports any Linux distribution. (String, Optional)
