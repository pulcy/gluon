PROJECT := gluon
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)
VERSION:= $(shell cat $(ROOTDIR)/VERSION)
COMMIT := $(shell git rev-parse --short HEAD)

GOBUILDDIR := $(SCRIPTDIR)/.gobuild
SRCDIR := $(SCRIPTDIR)
BINDIR := $(ROOTDIR)
VENDORDIR := $(ROOTDIR)/vendor

ORGPATH := github.com/pulcy
ORGDIR := $(GOBUILDDIR)/src/$(ORGPATH)
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)
BIN := $(BINDIR)/$(PROJECT)
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

update-vendor:
	@rm -Rf $(VENDORDIR)
	@pulcy go vendor -V $(VENDORDIR) \
		github.com/coreos/go-systemd/dbus \
		github.com/dchest/uniuri \
		github.com/juju/errgo \
		github.com/op/go-logging \
		github.com/spf13/cobra \
		github.com/spf13/pflag

$(BIN): $(GOBUILDDIR) $(GOBINDATA) $(SOURCES) templates/templates_bindata.go
	docker run \
		--rm \
		-v $(ROOTDIR):/usr/code \
		-e GO15VENDOREXPERIMENT=1 \
		-e GOPATH=/usr/code/.gobuild \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go build -a -ldflags "-X main.projectVersion=$(VERSION) -X main.projectBuild=$(COMMIT)" -o /usr/code/$(PROJECT) $(REPOPATH)

# Special rule, because this file is generated
templates/templates_bindata.go: $(TEMPLATES) $(GOBINDATA)
	$(GOBINDATA) -pkg templates -o templates/templates_bindata.go templates/
