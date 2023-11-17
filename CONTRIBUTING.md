# Contributing to pgxman

We welcome contributions to pgxman.

For information on how to build pgxman locally, please see [DEVELOPERS.md](DEVELOPERS.md).

### Report a bug

If the issue relates to a bug with installing a specific extension,
please open an issue at [pgxman/buildkit](https://github.com/pgxman/buildkit).

If you discover an issue in an extension itself, please report the bug directly
to the extension.

For pgxman CLI:

* Make sure you are on the latest version of pgxman
  * MacOS: Run `brew update` then `brew upgrade pgxman`
  * Linux: re-run the curl script to pull the latest version of pgxman
* Run and read `pgxman doctor`.
* Before submitting an issue, please search open issues.

### Propose a feature

Open an issue with a detailed description of your proposed feature, the motivation for it
and alternatives considered.

If you are interested in sending a PR for your feature, please let us know this in your issue.
We would be happy to discuss implementation details with you.

### Missing or incorrect documentation

We welcome issues or PRs for our documentation. Documentation is generated from the markdown
files in `docs`.

Note that `docs/cli` and `docs/man` are automatically generated from the code, so updates
to these files need to be made to their respective `internal/cmd/pgxman` files, then run `make docs`.

For information about previewing the docs locally, please see [docs/README.md](docs/README.md).

Thanks!
