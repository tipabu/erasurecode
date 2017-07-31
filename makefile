LIBECSRC=$(PWD)/deps/liberasurecode
ISALSRC=$(PWD)/deps/isa-l
BUILDDIR=$(PWD)/build

.PHONY: default test

default: $(BUILDDIR)/lib/liberasurecode.a $(BUILDDIR)/lib/libisal.a
	PKG_CONFIG_PATH=$(BUILDDIR)/lib/pkgconfig \
	go build

test: $(BUILDDIR)/lib/liberasurecode.a $(BUILDDIR)/lib/libisal.a
	DYLIB_LIBRARY_PATH=$(BUILDDIR)/lib \
	LD_LIBRARY_PATH=$(BUILDDIR)/lib \
	PKG_CONFIG_PATH=$(BUILDDIR)/lib/pkgconfig \
	go test -v

$(ISALSRC)/autogen.sh:
	git clone https://github.com/01org/isa-l.git $(ISALSRC)

$(LIBECSRC)/autogen.sh:
	git clone https://github.com/tipabu/liberasurecode.git $(LIBECSRC)

$(PWD)/deps/%/configure: $(PWD)/deps/%/autogen.sh
	cd $(@D) && ./autogen.sh

$(PWD)/deps/%/Makefile: $(PWD)/deps/%/configure
	cd $(@D) && ./configure --prefix=$(BUILDDIR)


$(BUILDDIR)/lib/liberasurecode.a: $(LIBECSRC)/Makefile
	cd $(LIBECSRC) && make install

$(BUILDDIR)/lib/libisal.a: $(ISALSRC)/Makefile
	cd $(ISALSRC) && make install
