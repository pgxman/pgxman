SHELL=/bin/bash -eo pipefail

.PHONY: build
build:
	test -f $(CURDIR)/buildkit/build && cd src && $(CURDIR)/buildkit/build || echo "No build script found"

.PHONY: install
install:
	test -f $(CURDIR)/buildkit/install && cd src && $(CURDIR)/buildkit/install || echo "No install script found"

.PHONY: clean
clean:
	test -f $(CURDIR)/buildkit/clean && cd src && $(CURDIR)/buildkit/clean || echo "No clean script found"
