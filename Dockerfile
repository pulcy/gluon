FROM alpine:3.3

ADD ./gluon /
ADD .build/etcd /etcd2
ADD .build/etcdctl /

ADD ./copy.sh /
RUN chmod +x /copy.sh

ENTRYPOINT ["/copy.sh"]
