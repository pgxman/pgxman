SHELL=/bin/bash -eo pipefail

.PHONY: build
build:

# Replace target-%v path with build-%v
SOURCEDIR = $(DESTDIR)-src
.PHONY: install
install:
	mkdir -p $(SOURCEDIR) && cp -r $(CURDIR)/src/. $(SOURCEDIR)/
	cd $(SOURCEDIR) && WORKDIR=$(SOURCEDIR) PG_VERSION=$$(echo "$(PG_CONFIG)" | sed -r "s/[^0-9]//g") bash $(CURDIR)/script/build

.PHONY: clean
clean:
