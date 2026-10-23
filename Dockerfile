FROM golang:1.26-bookworm AS builder

WORKDIR /sqldash

RUN apt-get update \
    && apt-get install -y --no-install-recommends make \
    && rm -rf /var/lib/apt/lists/*

ENV CGO_ENABLED=0

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN make build

FROM ghcr.io/tursodatabase/libsql-server:latest AS server

FROM postgres:18-bookworm@sha256:afc7e2d441324c0388fa80c3d24f733b4194a4eb7f47dd8ee2b08eb1a24a647c AS postgresaddons

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        postgresql-18=18.6-1.pgdg12+2 \
        postgresql-18-pgvector=0.8.7-1.pgdg12+1 \
        postgresql-18-rum=1.3.15-1.pgdg12+1 \
        postgresql-18-similarity=1.0-9.pgdg12+1 \
        postgresql-18-postgis-3=3.6.4+dfsg-2.pgdg12+1 \
        postgresql-18-postgis-3-scripts=3.6.4+dfsg-2.pgdg12+1 \
        postgresql-18-cron=1.6.8-1.pgdg12+2 \
        postgresql-18-repack=1.5.3-1.pgdg12+2 \
        postgresql-18-partman=5.5.0-1.pgdg12+1 \
        postgresql-18-squeeze=1.9.4-2.pgdg12+2 \
        postgresql-18-dirtyread=2.8-1.pgdg12+1 \
        postgresql-18-pg-stat-kcache=2.3.2-1.pgdg12+2 \
        postgresql-18-pg-qualstats=2.1.4-1.pgdg12+2 \
        postgresql-18-pg-hint-plan=1.8.0-3.pgdg12+1 \
        postgresql-18-pgaudit=18.0-3.pgdg12+1 \
        postgresql-18-ip4r=2.4.3-1.pgdg12+2 \
        postgresql-18-semver=0.42.0-1.pgdg12+1 \
        postgresql-18-roaringbitmap=1.2.0-1.pgdg12+1 \
        postgresql-18-timescaledb=2.30.2+dfsg-1.pgdg12+1 \
        postgresql-18-hll=2.21-1.pgdg12+2 \
    && rm -rf /var/lib/apt/lists/*

RUN set -eu; \
    mkdir -p /opt/postgresaddons/copyright /opt/postgresaddons/sharedir; \
    cp --archive --dereference /usr/share/postgresql/18/. /opt/postgresaddons/sharedir/; \
    if [ -n "$(find /opt/postgresaddons/sharedir -type l -print -quit)" ]; then \
        exit 1; \
    fi; \
    find /usr/lib/postgresql/18 -type f \( -name '*.so*' -o -executable \) \
        -exec ldd '{}' \; 2>/dev/null > /opt/postgresaddons/linked-libraries; \
    if grep -q 'not found' /opt/postgresaddons/linked-libraries; then \
        exit 1; \
    fi; \
    awk '$2 == "=>" && $3 ~ /^\// { print $3 } $1 ~ /^\// { print $1 }' \
        /opt/postgresaddons/linked-libraries | sort -u > /opt/postgresaddons/library-paths; \
    while IFS= read -r libraryPath; do \
        packageOwnership=$(dpkg-query --search "$libraryPath" 2>/dev/null \
            || dpkg-query --search "$(readlink -f "$libraryPath")"); \
        packageOwnership=$(printf '%s\n' "$packageOwnership" | awk 'NR == 1 { print; exit }'); \
        packageName=${packageOwnership%%: /*}; \
        dpkg-query --show --showformat='${binary:Package}=${Version}\n' "$packageName"; \
    done < /opt/postgresaddons/library-paths > /opt/postgresaddons/runtime-packages-unsorted; \
    dpkg-query --show --showformat='${binary:Package}=${Version}\n' \
        python3 python3-psycopg2 >> /opt/postgresaddons/runtime-packages-unsorted; \
    sort -u /opt/postgresaddons/runtime-packages-unsorted > /opt/postgresaddons/runtime-packages; \
    for copyrightPath in /usr/share/doc/postgresql-18-*/copyright; do \
        cp --parents --preserve=all "$copyrightPath" /opt/postgresaddons/copyright; \
    done

FROM postgres:18-bookworm@sha256:afc7e2d441324c0388fa80c3d24f733b4194a4eb7f47dd8ee2b08eb1a24a647c

COPY --from=postgresaddons /opt/postgresaddons/runtime-packages /tmp/postgres-runtime-packages

RUN apt-get update \
    && xargs --no-run-if-empty apt-get install -y --no-install-recommends < /tmp/postgres-runtime-packages \
    && rm -rf /var/lib/apt/lists/* /tmp/postgres-runtime-packages

COPY --from=postgresaddons /usr/lib/postgresql/18/ /usr/lib/postgresql/18/
COPY --from=postgresaddons /opt/postgresaddons/sharedir/ /usr/share/postgresql/18/
COPY --from=postgresaddons /opt/postgresaddons/copyright/ /

WORKDIR /sqldash

COPY --from=server /bin/sqld /usr/local/bin/sqld
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /sqldash/bin/sqldash .
COPY templates ./templates
COPY static ./static

ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt \
    SQLDASH_DATA_DIRECTORY=/data \
    SQLDASH_HTTP_PORT=80 \
    SQLDASH_HTTPS_PORT=443 \
    SQLDASH_SQLD_BINARY=/usr/local/bin/sqld \
    SQLDASH_POSTGRES_ENABLED=true \
    SQLDASH_POSTGRES_BINARY_DIRECTORY=/usr/lib/postgresql/18/bin \
    PGDATA=/data/postgres/18/data

VOLUME /data

EXPOSE 80 443 5432

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 CMD ["/sqldash/sqldash", "health"]

STOPSIGNAL SIGTERM
ENTRYPOINT ["/sqldash/sqldash"]
CMD []
