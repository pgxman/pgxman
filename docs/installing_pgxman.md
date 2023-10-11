## Installation

PGXMan is compatible with the following Debian-based Linux distributions:

- [Debian Bookworm](https://www.debian.org/releases/bookworm)
- [Ubuntu Jammy](https://releases.ubuntu.com/jammy)

### Prerequisites

- Apt Package Manager
- [PostgreSQL](installing_postgres.md)
- If you plan on [building an extension](building_an_extension.md):
  - Docker

### Installer

The simplest method to install `pgxman` is through the installer script:

```console
curl -sfL https://github.com/pgxman/release/releases/latest/download/install.sh | sh -
```

### Manual Download

Download the latest [compiled Linux
binaries](https://github.com/pgxman/release/releases/) and add them to your
system's executable path.
