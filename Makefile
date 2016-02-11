PROJECT := yard
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)
VERSION:= $(shell cat $(ROOTDIR)/VERSION)
COMMIT := $(shell git rev-parse --short HEAD)

GOBUILDDIR := $(SCRIPTDIR)/.gobuild
SRCDIR := $(SCRIPTDIR)
BINDIR := $(ROOTDIR)
VENDORDIR := $(ROOTDIR)/vendor

ORGPATH := arvika.pulcy.com/pulcy
ORGDIR := $(GOBUILDDIR)/src/$(ORGPATH)
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)
BIN := $(BINDIR)/$(PROJECT)
BINGPG := $(BIN).gpg
GOBINDATA := $(GOBUILDDIR)/bin/go-bindata

GOPATH := $(GOBUILDDIR)
GOVERSION := 1.5.3

ifndef GOOS
	GOOS := linux
endif
ifndef GOARCH
	GOARCH := amd64
endif

SOURCES := $(shell find $(SRCDIR) -name '*.go')
TEMPLATES := $(shell find $(SRCDIR) -name '*.tmpl')

.PHONY: all clean deps

all: $(BIN) $(BINGPG)

clean:
	rm -Rf $(BIN) $(BINGPG) $(GOBUILDDIR)

deps:
	@${MAKE} -B -s $(GOBUILDDIR) $(GOBINDATA)

$(GOBINDATA):
	GOPATH=$(GOPATH) go get github.com/jteeuwen/go-bindata/...

$(GOBUILDDIR):
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s ../../../.. $(REPODIR)
	@cd $(GOPATH) && pulcy go get github.com/coreos/go-systemd/dbus
	@cd $(GOPATH) && pulcy go get github.com/dchest/uniuri
	@cd $(GOPATH) && pulcy go get github.com/juju/errgo
	@cd $(GOPATH) && pulcy go get github.com/op/go-logging
	@cd $(GOPATH) && pulcy go get github.com/spf13/cobra
	@cd $(GOPATH) && pulcy go get github.com/spf13/pflag


$(BIN): $(GOBUILDDIR) $(GOBINDATA) $(SOURCES) templates/templates_bindata.go
	docker run \
	    --rm \
	    -v $(ROOTDIR):/usr/code \
	    -e GOPATH=/usr/code/.gobuild \
	    -e GOOS=$(GOOS) \
	    -e GOARCH=$(GOARCH) \
	    -w /usr/code/ \
	    golang:$(GOVERSION) \
	    go build -a -ldflags "-X main.projectVersion=$(VERSION) -X main.projectBuild=$(COMMIT)" -o /usr/code/$(PROJECT)

$(BINGPG): $(BIN)
	@sh -c 'if [ -z $(YARD_PASSPHRASE) ]; then echo YARD_PASSPHRASE missing && exit 1; fi'
	@rm -Rf $(BINGPG)
	@docker run \
	    --rm \
		-t \
	    -v $(ROOTDIR):/usr/code \
		-e YARD_PASSPHRASE=$(YARD_PASSPHRASE) \
	    -w /usr/code/ \
		pulcy/gpg \
		--batch --armor --output yard.gpg --passphrase $(YARD_PASSPHRASE) --symmetric yard

# Special rule, because this file is generated
templates/templates_bindata.go: $(TEMPLATES) $(GOBINDATA)
	$(GOBINDATA) -pkg templates -o templates/templates_bindata.go templates/
