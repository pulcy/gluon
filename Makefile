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
GOVERSION := 1.10.0-alpine

ETCDVERSION := v3.1.5

RKTVERSION := v1.22.0

CONSULVERSION := 0.7.2
CONSULTEMPLATEVERSION := 0.16.0

K8SVERSION := v1.5.1

ifndef GOOS
	GOOS := linux
endif
ifndef GOARCH
	GOARCH := amd64
endif

SOURCES := $(shell find $(SRCDIR) -name '*.go')
TEMPLATES := $(shell find $(SRCDIR)/templates -name '*')

.PHONY: all clean deps consul weave rkt etcd kubernetes cni

all: .build/certdump consul kubernetes weave rkt $(BIN) $(BINGPG) etcd cni

clean:
	rm -Rf $(BIN) $(BINGPG) $(GOBUILDDIR) .build 

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
	$(GOBINDATA) -pkg templates -o templates/templates_bindata.go templates/...

.build/certdump: certdump/certdump.go 
	mkdir -p .build
	docker run \
		--rm \
		-v $(ROOTDIR)/certdump/:/usr/code \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go build -o /usr/code/certdump 
	mv $(ROOTDIR)/certdump/certdump .build/

# ETCD 

etcd: .build/etcd

.build/etcd: .build/etcd.tar.gz
	cd .build && tar zxf etcd.tar.gz && cp etcd-${ETCDVERSION}-linux-amd64/etcd* . && touch ./etcd

.build/etcd.tar.gz:
	mkdir -p .build
	curl -L  https://github.com/coreos/etcd/releases/download/$(ETCDVERSION)/etcd-$(ETCDVERSION)-linux-amd64.tar.gz -o .build/etcd.tar.gz

# Rkt
rkt: .build/rkt

.build/rkt: .build/rkt.tar.gz
	cd .build && tar zxf rkt.tar.gz && mv rkt-${RKTVERSION} rkt && touch ./rkt/rkt
	cp .build/rkt/init/systemd/tmpfiles.d/rkt.conf templates/rkt/

.build/rkt.tar.gz:
	mkdir -p .build
	curl -L  https://github.com/coreos/rkt/releases/download/$(RKTVERSION)/rkt-$(RKTVERSION).tar.gz -o .build/rkt.tar.gz

# Weave 

weave: .build/weave

.build/weave:
	mkdir -p .build
	curl -L https://github.com/weaveworks/weave/releases/download/v1.9.3/weave -o .build/weave

# Consul 

consul: .build/consul .build/consul-template

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

# Kubernetes 

kubernetes: .build/kubectl .build/kubelet .build/kube-proxy

.build/kubectl:
	@mkdir -p .build
	@curl -L https://storage.googleapis.com/kubernetes-release/release/$(K8SVERSION)/bin/linux/amd64/kubectl -o .build/kubectl

.build/kubelet:
	@mkdir -p .build
	@curl -L https://storage.googleapis.com/kubernetes-release/release/$(K8SVERSION)/bin/linux/amd64/kubelet -o .build/kubelet

.build/kube-proxy:
	@mkdir -p .build
	@curl -L https://storage.googleapis.com/kubernetes-release/release/$(K8SVERSION)/bin/linux/amd64/kube-proxy -o .build/kube-proxy

# CNI 

cni: .build/cni 

.build/cni: .build/cni.tar.gz
	@mkdir -p .build/cni
	@tar -xvf .build/cni.tar.gz -C .build/cni

.build/cni.tar.gz: 
	@mkdir -p .build
	@curl -L https://storage.googleapis.com/kubernetes-release/network-plugins/cni-07a8a28637e97b22eb8dfe710eeae1344f69d16e.tar.gz -o .build/cni.tar.gz

