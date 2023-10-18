# pgxman Developer Guide

We welcome contributions to pgxman. Here's what you'll need to get started.

## Install prerequisites

* Go 1.21 or later. Install from homebrew, asdf, <https://go.dev/dl/>, etc.
* make
* git
* docker

## Compile

```console
make build
```

## Test

There's several test suites:

```console
make test      # unit tests
make e2etest   # end-to-end (integration) tests
make vet       # linter
```

## Install

```console
make install
```

## Release

The current process is to use `git tag` to check for the most recent version
number, then choose a version number as appropriate:

```console
script/release 1.2.3
```
