FROM alpine:3.11

RUN apk add s3fs-fuse --no-cache --repository http://dl-3.alpinelinux.org/alpine/edge/testing/ --allow-untrusted

COPY s3vol-amd64 /usr/local/bin/s3vol

ENTRYPOINT [ "/usr/local/bin/s3vol" ]

CMD [ "serve" ]