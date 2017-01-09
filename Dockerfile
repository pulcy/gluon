FROM alpine:3.4

ADD .build/etcd /dist/etcd2
ADD .build/etcdctl /dist/

ADD .build/fleetd /dist/
ADD .build/fleetctl /dist/

ADD .build/rkt/rkt /dist/rkt/
COPY .build/rkt/stage1-coreos.aci /dist/rkt/
COPY .build/rkt/scripts/setup-data-dir.sh /dist/rkt/

COPY .build/weave /dist/

COPY .build/consul /dist/
COPY .build/consul-template /dist/

COPY .build/certdump /dist/

COPY .build/kube* /dist/

ADD ./copy.sh /
RUN chmod +x /copy.sh

ADD ./gluon /dist/

ENTRYPOINT ["/copy.sh"]
