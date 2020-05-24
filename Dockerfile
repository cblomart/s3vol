FROM alpine:3.11

ARG S3FS_VERSION=v1.86

RUN apk --update add bash fuse libcurl libxml2 libstdc++ libgcc alpine-sdk automake autoconf libxml2-dev fuse-dev curl-dev git; \
git clone https://github.com/s3fs-fuse/s3fs-fuse.git; \
cd s3fs-fuse; \
git checkout tags/${S3FS_VERSION}; \
./autogen.sh; \
./configure --prefix=/usr; \
make; \
make install; \
apk del alpine-sdk automake autoconf libxml2-dev fuse-dev curl-dev git; \
rm -rf /var/cache/apk/*;

COPY s3vol-amd64 /usr/local/bin/s3vol

ENTRYPOINT [ "/usr/local/bin/s3vol" ]

CMD [ "serve" ]