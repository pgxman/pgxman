# pgxman.yaml

A `pgxman.yaml` is a YAML configuration file used to declare a collection of PostgreSQL extensions for installation via PGXMan.
It serves as an input file for the command `pgxman install -f /PATH_TO/pgxman.yaml`, defining the required extensions, their versions,
and the targeting PostgreSQL versions.

## Fields

### `apiVersion`

- **Description**: Defines the API version to ensure compatibility with PGXMan.
- **Type**: String
- **Required**: Yes
- **Supported Values**: As of now, only `v1` is supported.

### `extensions`

- **Description**: Lists the extensions to be installed.
- **Type**: List of objects
- **Required**: No
- **Object Fields**:
  - `name`:
    - **Description**: Specifies the name of the extension. Either the `name` or the `path` field must be present.
    - **Type**: String
    - **Required**: No
  - `version`:
    - **Description**: Specifies the version of the extension. This field is mandatory if the `name` field is provided.
    - **Type**: String
    - **Required**: Yes if `name` is present
  - `path`:
    - **Description**: Specifies the local path to the extension package. Either the `name` or the `path` field must be present.
    - **Type**: String
    - **Required**: No
  - `options`:
    - **Description**: Specifies the options to be passed to corresponding package manager when installing the extension.
    - **Type**: List of strings
    - **Required**: No

### `pgVersions`

- **Description**: Specifies the PostgreSQL versions that the extensions are targeting.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `"13"`, `"14"`, `"15"`

## Example

Here's an example that illustrates the usage of these fields:

```yaml
apiVersion: v1
extensions:
  - name: "pgvector"
    version: "0.4.4"
  - path: "/local/path/to/extension"
pgVersions:
  - "14"
  - "15"
```

This example file installs the `pgvector` extension with version `0.4.4`, and another extension located at a specific local path, both for PostgreSQL versions 14 and 15.
