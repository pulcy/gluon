FROM alpine:3.2

ADD ./yard /yard
ADD ./copy.sh /copy.sh
RUN chmod +x /copy.sh

ENTRYPOINT ["/copy.sh"]

