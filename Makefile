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
GOVERSION := 1.7.0-alpine

ETCDVERSION := v3.0.1

FLEETVERSION := v0.11.7
FLEETBUILDDIR := $(ROOTDIR)/.build/fleet

ifndef GOOS
	GOOS := linux
endif
ifndef GOARCH
	GOARCH := amd64
endif

SOURCES := $(shell find $(SRCDIR) -name '*.go')
TEMPLATES := $(shell find $(SRCDIR) -name '*.tmpl')

.PHONY: all clean deps

all: $(BIN) $(BINGPG) .build/etcd .build/fleetd

clean:
	rm -Rf $(BIN) $(BINGPG) $(GOBUILDDIR) .build $(FLEETBUILDDIR)

deps:
	@${MAKE} -B -s $(GOBUILDDIR) $(GOBINDATA)

$(GOBINDATA):
	GOPATH=$(GOPATH) go get github.com/jteeuwen/go-bindata/...

$(GOBUILDDIR):
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s ../../../.. $(REPODIR)

update-vendor:
	@rm -Rf $(VENDORDIR)
	@pulsar go vendor -V $(VENDORDIR) \
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
		-e GOPATH=/usr/code/.gobuild \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e CGO_ENABLED=0 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go build -a -installsuffix netgo -tags netgo -ldflags "-X main.projectVersion=$(VERSION) -X main.projectBuild=$(COMMIT)" -o /usr/code/$(PROJECT) $(REPOPATH)

# Special rule, because this file is generated
templates/templates_bindata.go: $(TEMPLATES) $(GOBINDATA)
	$(GOBINDATA) -pkg templates -o templates/templates_bindata.go templates/

.build/etcd: .build/etcd.tar.gz
	cd .build && tar zxf etcd.tar.gz && cp etcd-${ETCDVERSION}-linux-amd64/etcd* . && touch ./etcd

.build/etcd.tar.gz:
	mkdir -p .build
	curl -L  https://github.com/coreos/etcd/releases/download/$(ETCDVERSION)/etcd-$(ETCDVERSION)-linux-amd64.tar.gz -o .build/etcd.tar.gz

.build/fleetd: $(FLEETBUILDDIR)
	docker run \
		--rm \
		-v $(FLEETBUILDDIR):/usr/code \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e CGO_ENABLED=0 \
		-w /usr/code/ \
		golang:1.7.0 \
		/usr/code/build
	cp $(FLEETBUILDDIR)/bin/fleetd .build/
	cp $(FLEETBUILDDIR)/bin/fleetctl .build/

$(FLEETBUILDDIR):
	@pulsar get -b $(FLEETVERSION) https://github.com/coreos/fleet.git $(FLEETBUILDDIR)
