# How It Works

In short:

* buildkits contain metadata about how to build packages.
* every time a buildkit is created or updated, we build packages which are
  added to a repository.
* `pgxman install` adds pgxman's repository to your system, and then installs
  the packages using your system package manager.

## Supported systems

pgxman's build system works with your local package manager. Currently, pgxman
supports `apt`-based systems and targets [Debian
Bookworm](https://www.debian.org/releases/bookworm) and [Ubuntu
Jammy](https://releases.ubuntu.com/jammy). In the future, other package
managers and operating system releases will be supported. Currently, we are
targeting:

- [x] apt
- [ ] brew
- [ ] rpm

## Installation

When installing an extension, pgxman's package repository is added to your
system in order to install the packages into your system. This way, pgxman is
able to handle dependency management, installation, and uninstallation through
your system's package manager.

## Buildkits

pgxman uses [a repository of
buildkits](https://github.com/pgxman/buildkit/tree/main/buildkit) to know what
extensions are available. When you use `search` or `install`, the buildkit
metadata is used to obtain information about that extension. A cached copy
of this repository is stored in the `pgxman` folder in your [user config
directory](https://pkg.go.dev/os#UserConfigDir).

Each buildkit specifies how to build each extension, and the buildkit build
system builds it for each package manager. When a buildkit is added or updated,
a build is conducted automatically (using Github Actions) and the packages are
stored in pgxman's repository.

## Building a buildkit

`pgxman build` uses Docker to build the buildkit packages locally. The result
is placed into the `out/` directory in your current working directory.
