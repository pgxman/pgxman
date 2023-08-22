# Buildkit Specification

PGXMan buildkit is a configuration file in YAML format that `pgxman` uses to specify how a PostgreSQL extension should be built and packaged.

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
- **Fields**:
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

## `runDependencies`

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

## `builders`

- **Description**: Specify the builders to be used. If not provided, all supported builders are used.
- **Type**: Object
- **Required**: No
- **Fields**:
  - `debian:bookworm`:
    - **Description**: Specifies the Debian Bookworm builder.
    - **Type**: Object
    - **Required**: No
    - **Fields**:
      - `buildDependencies`:
        - **Description**: Lists the packages necessary for building for the `debian:bookworm` builder. This list overrides the global `buildDependencies`.
        - **Type**: List of strings
        - **Required**: No
      - `runDependencies`:
        - **Description**: Lists the rpackages needed for the extension to function properly at runtime for the `debian:bookworm` builder. This list overrides the global `runDependencies`.
        - **Type**: List of strings
        - **Required**: No
      - `image`:
        - **Description** Speifies the build image.
        - **Type**: String
        - **Required**: No
        - **Default Value**: `ghcr.io/pgxman/builder/debian/bookworm`
      - `aptRepositories`:
        - **Description**: Lists the APT repositories containing the above Debian packages.
        - **Type**: List of objects
        - **Required**: No
        - **Fields**:
          - `id`:
            - **Description**: The repository's id.
            - **Type**: String
            - **Required**: Yes
          - `types`:
            - **Description**: The repository's types.
            - **Type**: String
            - **Required**: Yes
            - **Supported Values** `deb`, `deb-src`
          - `uris`:
            - **Description**: The repository's URIs.
            - **Type**: List of strings
            - **Required**: Yes
          - `suites`:
            - **Description**: The repository's suites within the APT repository.
            - **Type**: List of objects
            - **Required**: Yes
          - `components`:
            - **Description**: The components of the APT repository.
            - **Type**: List of strings
            - **Required**: Yes
          - `signedKey`:
            - **Description**: The GPG signed key of the APT repository.
            - **Type**: List of objects
            - **Required**: Yes
            - **Fields**:
              - `uri`:
                - **Description**: The HTTP URL to download the GPG key.
                - **Type**: String
                - **Required**: Yes
              - `format`:
                - **Description**: The format of the GPG key.
                - **Type**: String
                - **Required**: Yes
                - **Supported Values**: `gpg`, `asc`
  - `ubuntu:jammy`:
    - **Description**: Specifies the Ubuntu Jammy builder.
    - **Type**: Object
    - **Required**: No
    - **Fields**:
      - `buildDependencies`:
        - **Description**: Lists the packages necessary for building for the `ubuntu:jammy` builder. This list overrides the global `buildDependencies`.
        - **Type**: List of strings
        - **Required**: No
      - `runDependencies`:
        - **Description**: Lists the rpackages needed for the extension to function properly at runtime for the `ubuntu:jammy` builder. This list overrides the global `runDependencies`.
        - **Type**: List of strings
        - **Required**: No
      - `image`:
        - **Description** Speifies the build image.
        - **Type**: String
        - **Required**: No
        - **Default Value**: `ghcr.io/pgxman/builder/ubuntu/jammy`
      - `aptRepositories`:
        - **Description**: Lists the APT repositories containing the above Debian packages.
        - **Type**: List of objects
        - **Required**: No
        - **Fields**:
          - `id`:
            - **Description**: The repository's id.
            - **Type**: String
            - **Required**: Yes
          - `types`:
            - **Description**: The repository's types.
            - **Type**: String
            - **Required**: Yes
            - **Supported Values** `deb`, `deb-src`
          - `uris`:
            - **Description**: The repository's URIs.
            - **Type**: List of strings
            - **Required**: Yes
          - `suites`:
            - **Description**: The repository's suites within the APT repository.
            - **Type**: List of objects
            - **Required**: Yes
          - `components`:
            - **Description**: The components of the APT repository.
            - **Type**: List of strings
            - **Required**: Yes
          - `signedKey`:
            - **Description**: The GPG signed key of the APT repository.
            - **Type**: List of objects
            - **Required**: Yes
            - **Fields**:
              - `uri`:
                - **Description**: The HTTP URL to download the GPG key.
                - **Type**: String
                - **Required**: Yes
              - `format`:
                - **Description**: The format of the GPG key.
                - **Type**: String
                - **Required**: Yes
                - **Supported Values**: `gpg`, `asc`
