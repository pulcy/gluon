FROM alpine:3.4

ADD .build/etcd /dist/etcd2
ADD .build/etcdctl /dist/
ADD .build/fleetd /dist/
ADD .build/fleetctl /dist/
ADD .build/rkt/rkt /dist/rkt/
COPY .build/rkt/stage1-coreos.aci /dist/rkt/
COPY .build/rkt/scripts/setup-data-dir.sh /dist/rkt/
COPY .build/weave /dist/

ADD ./copy.sh /
RUN chmod +x /copy.sh

ADD ./gluon /dist/

ENTRYPOINT ["/copy.sh"]
