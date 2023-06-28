SHELL=/bin/bash -eo pipefail

.PHONY: build
build:

# Replace target-%v path with build-%v
SOURCEDIR = $(subst target,build,$(DESTDIR))
.PHONY: install
install:
	mkdir -p $(SOURCEDIR) && tar xf $(CURDIR)/../pgvector_0.4.2.orig.tar.gz -C $(SOURCEDIR) --strip-components 1
	cd $(SOURCEDIR) && $(CURDIR)/buildkit/install

.PHONY: clean
clean:
