SHELL=/bin/bash -eo pipefail

.PHONY: build
build:

# Replace target-%v path with build-%v
SOURCEDIR = $(DESTDIR)-src
.PHONY: install
install:
	mkdir -p $(SOURCEDIR) && cp -r $(CURDIR)/src/. $(SOURCEDIR)/
	cd $(SOURCEDIR) && bash $(CURDIR)/script/build

.PHONY: clean
clean:
