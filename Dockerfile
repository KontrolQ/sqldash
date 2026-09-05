FROM golang:1.26-alpine AS builder

WORKDIR /sqldash

RUN apk add --no-cache make

ENV CGO_ENABLED=0

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN make build

FROM ghcr.io/tursodatabase/libsql-server:latest AS server

FROM gcr.io/distroless/cc-debian12

WORKDIR /sqldash

COPY --from=server /bin/sqld /usr/local/bin/sqld
COPY --from=builder /sqldash/bin/sqldash .
COPY templates ./templates
COPY static ./static

ENV SQLDASH_DATA_DIRECTORY=/data \
    SQLDASH_HTTP_PORT=80 \
    SQLDASH_HTTPS_PORT=443 \
    SQLDASH_SQLD_BINARY=/usr/local/bin/sqld

VOLUME /data

EXPOSE 80 443

CMD ["./sqldash"]
