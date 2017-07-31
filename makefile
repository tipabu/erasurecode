LIBECSRC=$(PWD)/deps/liberasurecode
ISALSRC=$(PWD)/deps/isa-l
GFCOMPLETESRC=$(PWD)/deps/gf-complete
JERASURESRC=$(PWD)/deps/jerasure
BUILDDIR=$(PWD)/build

.PHONY: default test

default: $(BUILDDIR)/lib/liberasurecode.a $(BUILDDIR)/lib/libisal.a $(BUILDDIR)/lib/libJerasure.la
	PKG_CONFIG_PATH=$(BUILDDIR)/lib/pkgconfig \
	go build

test: $(BUILDDIR)/lib/liberasurecode.a $(BUILDDIR)/lib/libisal.a $(BUILDDIR)/lib/libJerasure.la
	DYLIB_LIBRARY_PATH=$(BUILDDIR)/lib \
	LD_LIBRARY_PATH=$(BUILDDIR)/lib \
	PKG_CONFIG_PATH=$(BUILDDIR)/lib/pkgconfig \
	go test -v

$(ISALSRC)/autogen.sh:
	git clone https://github.com/01org/isa-l.git $(ISALSRC)

$(LIBECSRC)/autogen.sh:
	git clone https://github.com/tipabu/liberasurecode.git $(LIBECSRC)

$(JERASURESRC)/configure.ac:
	git clone http://lab.jerasure.org/jerasure/jerasure.git $(JERASURESRC)

$(GFCOMPLETESRC)/autogen.sh:
	git clone http://lab.jerasure.org/jerasure/gf-complete.git $(GFCOMPLETESRC)

$(PWD)/deps/%/configure: $(PWD)/deps/%/autogen.sh
	cd $(@D) && ./autogen.sh

$(JERASURESRC)/configure: $(JERASURESRC)/configure.ac
	cd $(@D) && autoreconf --force --install

$(PWD)/deps/%/Makefile: $(PWD)/deps/%/configure
	cd $(@D) && ./configure --prefix=$(BUILDDIR) \
	LDFLAGS=-L$(BUILDDIR)/lib CFLAGS=-I$(BUILDDIR)/include


$(BUILDDIR)/lib/liberasurecode.a: $(LIBECSRC)/Makefile
	cd $(LIBECSRC) && make install

$(BUILDDIR)/lib/libisal.a: $(ISALSRC)/Makefile
	cd $(ISALSRC) && make install

$(BUILDDIR)/lib/libgf_complete.a: $(GFCOMPLETESRC)/Makefile
	cd $(GFCOMPLETESRC) && make install

$(BUILDDIR)/lib/libJerasure.la: $(BUILDDIR)/lib/libgf_complete.a $(JERASURESRC)/Makefile
	cd $(JERASURESRC) && make install
