LIBECSRC=$(PWD)/deps/liberasurecode
ISALSRC=$(PWD)/deps/isa-l
BUILDDIR=$(PWD)/build

.PHONY: default test

default: $(BUILDDIR)/lib/liberasurecode-1.so
	PKG_CONFIG_PATH=$(BUILDDIR)/lib/pkgconfig go build

test: $(BUILDDIR)/lib/liberasurecode-1.so
	DYLIB_LIBRARY_PATH=$(BUILDDIR)/lib LD_LIBRARY_PATH=$(BUILDDIR)/lib PKG_CONFIG_PATH=$(BUILDDIR)/lib/pkgconfig go test

$(LIBECSRC)/autogen.sh:
	git clone https://github.com/tipabu/liberasurecode.git $(LIBECSRC)

$(LIBECSRC)/configure: $(LIBECSRC)/autogen.sh
	cd $(LIBECSRC) && ./autogen.sh

$(LIBECSRC)/Makefile: $(LIBECSRC)/configure
	cd $(LIBECSRC) && ./configure --prefix=$(BUILDDIR)

$(BUILDDIR)/lib/liberasurecode-1.so: $(LIBECSRC)/Makefile
	cd $(LIBECSRC) && make install
