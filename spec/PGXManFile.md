# PGXManFile

A `PGXManFile` is a YAML configuration file used to declare a collection of PostgreSQL extensions for installation via PGXMan.
It serves as an input file for the command `pgxman install -f PGXManFile`, defining the required extensions, their versions,
and the targeting PostgreSQL versions.

## Fields

### `apiVersion`

- **Description**: Defines the API version of the PGXManFile to ensure compatibility with PGXMan.
- **Type**: String
- **Required**: Yes
- **Supported Values**: As of now, only `v1` is supported.

### `extensions`

- **Description**: Lists the extensions to be installed.
- **Type**: List of objects
- **Required**: No
- **Object Fields**:
  - `name`: Specifies the name of the extension. Either the `name` or the `path` field must be present. (String, Optional)
  - `version`: Specifies the version of the extension. This field is mandatory if the `name` field is provided. (String, Required if `name` is present)
  - `path`: Provides the local path to the extension package. Either the `name` or the `path` field must be present. (String, Optional)

### `pgVersions`

- **Description**: Specifies the PostgreSQL versions that the extensions are targeting at.
- **Type**: List of strings
- **Required**: No
- **Supported Values**: `"13"`, `"14"`, `"15"`

## Example

Here's an example of a PGXManFile that illustrates the usage of these fields:

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
