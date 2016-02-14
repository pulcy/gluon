FROM alpine:3.3

ADD ./gluon /
ADD ./copy.sh /
RUN chmod +x /copy.sh

ENTRYPOINT ["/copy.sh"]
