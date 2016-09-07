FROM alpine:3.3

ADD ./gluon /
ADD .build/etcd /etcd2
ADD .build/etcdctl /
ADD .build/fleetd /
ADD .build/fleetctl /
ADD .build/rkt /
ADD .build/stage1-coreos.aci /

ADD ./copy.sh /
RUN chmod +x /copy.sh

ENTRYPOINT ["/copy.sh"]
