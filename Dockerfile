FROM alpine:3.2

RUN apk add -U gpgme \
&& rm -rf /var/cache/apk/*

ADD ./yard.gpg /
ADD ./copy.sh /
RUN chmod +x /copy.sh

ENTRYPOINT ["/copy.sh"]

