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
GOVERSION := 1.7.3-alpine

ETCDVERSION := v3.0.15

FLEETVERSION := e6c838b9bdc0184d727eb8c43bd856ecfa4a1519
FLEETBUILDDIR := $(ROOTDIR)/.build/fleet

RKTVERSION := v1.21.0

CONSULVERSION := 0.7.2
CONSULTEMPLATEVERSION := 0.16.0

ifndef GOOS
	GOOS := linux
endif
ifndef GOARCH
	GOARCH := amd64
endif

SOURCES := $(shell find $(SRCDIR) -name '*.go')
TEMPLATES := $(shell find $(SRCDIR)/templates -name '*')

.PHONY: all clean deps

all: .build/certdump .build/consul .build/consul-template .build/weave .build/rkt $(BIN) $(BINGPG) .build/etcd .build/fleetd

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
		github.com/spf13/pflag \
		golang.org/x/sync/errgroup

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

.build/certdump: certdump/certdump.go 
	mkdir -p .build
	docker run \
		--rm \
		-v $(ROOTDIR)/certdump/:/usr/code \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-w /usr/code/ \
		golang:1.7.4 \
		go build -o /usr/code/certdump 
	mv $(ROOTDIR)/certdump/certdump .build/

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

.build/rkt: .build/rkt.tar.gz
	cd .build && tar zxf rkt.tar.gz && mv rkt-${RKTVERSION} rkt && touch ./rkt/rkt
	cp .build/rkt/init/systemd/tmpfiles.d/rkt.conf templates/

.build/rkt.tar.gz:
	mkdir -p .build
	curl -L  https://github.com/coreos/rkt/releases/download/$(RKTVERSION)/rkt-$(RKTVERSION).tar.gz -o .build/rkt.tar.gz

.build/weave:
	mkdir -p .build
	curl -L git.io/weave -o .build/weave

.build/consul: .build/consul.zip
	@rm -f .build/consul
	@unzip .build/consul.zip -d .build 
	@touch .build/consul

.build/consul.zip:
	@mkdir -p .build
	@curl -L https://releases.hashicorp.com/consul/$(CONSULVERSION)/consul_$(CONSULVERSION)_linux_amd64.zip -o .build/consul.zip

.build/consul-template: .build/consul-template.zip
	@rm -f .build/consul-template
	@unzip .build/consul-template.zip -d .build 
	@touch .build/consul-template

.build/consul-template.zip:
	@mkdir -p .build
	@curl -L https://releases.hashicorp.com/consul-template/$(CONSULTEMPLATEVERSION)/consul-template_$(CONSULTEMPLATEVERSION)_linux_amd64.zip -o .build/consul-template.zip 
