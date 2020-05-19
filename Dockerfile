FROM golang:1.14-alpine3.11 AS builder

RUN apk add --no-cache gcc musl-dev upx

WORKDIR /app

COPY . .

RUN go build ./cmd/s3vol.go

RUN upx -qq s3vol

FROM alpine:3.11

RUN apk add s3fs-fuse --no-cache --repository http://dl-3.alpinelinux.org/alpine/edge/testing/ --allow-untrusted

COPY --from=builder /app/s3vol /usr/local/bin/s3vol

ENTRYPOINT [ "/usr/local/bin/s3vol" ]

CMD [ "serve" ]