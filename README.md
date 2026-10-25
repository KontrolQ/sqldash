# sqldash

Self-hosted control panel for libSQL, PostgreSQL, and Redis. Create databases, manage connections, browse and edit data, run SQL or Redis commands, import/export backups, inspect activity.

Each database keeps its engine; there is no cross-engine conversion.

## Run the container

```sh
docker build -t sqldash .
docker run --name sqldash \
  -p 80:80 -p 443:443 -p 7070:7070 -p 5432:5432 -p 6379:6379 \
  -e SQLDASH_DOMAIN=databases.example.com \
  -v sqldash-data:/data \
  --stop-timeout 70 \
  sqldash
```

Open `http://localhost` (or your domain), create the operator account. That account owns the installation.

Keep `/data` persistent. Run **one container per data volume**, stop the old container before starting its replacement.

## Engines

- **libSQL:** one shared server; databases are namespaces. Use its HTTP URL and an application token.
- **PostgreSQL:** one 18 cluster; each database gets an application login whose username is exactly the database name. Native gateway on port 5432.
- **Redis:** one standalone process per database, DB 0 only. Primary application username is exactly the database name, matching PostgreSQL. Native gateway on port 6379.

Native gateways are not SQL parsers; they forward authenticated client traffic and enforce per-credential database/instance policy.

## Development

```sh
make dev
```

Runs `compose.dev.yml` (dashboard on `http://localhost:8888`, libSQL backend private on its Docker network). Set `SQLDASH_DEVELOPMENT_LIBSQL_VOLUME` in `.env` to the backend's existing Docker volume. `make dev-stop` stops both services without deleting volumes.

## Settings

All settings use the `SQLDASH_` prefix; env vars take precedence over `.env` in the working directory.

| Setting | Default | Purpose |
|---|---|---|
| `DOMAIN` | `localhost` | Database hostname base and default native host |
| `PUBLIC_URL` | unset | Public dashboard origin for invitation/recovery links |
| `WEB_ADDRESS` | `127.0.0.1:7070` | Dashboard web worker listener |
| `HTTP_PORT` / `HTTPS_PORT` | `8800` / `8443` | HTTP proxy ports (image uses 80/443) |
| `REDIRECT_TO_TLS` | `true` | Redirect HTTP to HTTPS |
| `DEBUG` | `false` | Development diagnostics |
| `MAXIMUM_IMAGE_UPLOAD_BYTES` | `10485760` | Profile-picture limit |
| `DATA_DIRECTORY` | `./data` | Persistent root (image uses `/data`) |
| `SQLD_MANAGED` | `true` | Supervise libSQL |
| `SQLD_BINARY` | `sqld` | libSQL executable |
| `SQLD_ADDRESS` | `127.0.0.1:8080` | Private libSQL endpoint |
| `SQLD_ADMIN_ADDRESS` | `127.0.0.1:8081` | Private libSQL administration |
| `POSTGRES_ENABLED` | `true` | Make PostgreSQL available |
| `POSTGRES_MANAGED` | `true` | Supervise the dedicated PostgreSQL cluster |
| `POSTGRES_BINARY_DIRECTORY` | `/usr/lib/postgresql/18/bin` | Server and native client tools |
| `POSTGRES_ADDRESS` | `127.0.0.1:55432` | Private PostgreSQL endpoint |
| `POSTGRES_GATEWAY_ADDRESS` | `:5432` | Native listener |
| `POSTGRES_GATEWAY_ENABLED` | `true` | Run native PostgreSQL ingress |
| `POSTGRES_PUBLIC_HOST` | unset | Advertised native hostname |
| `POSTGRES_PUBLIC_PORT` | `5432` | Advertised external port |
| `POSTGRES_ADMIN_USERNAME` | `sqldash_admin` | Private management principal |
| `POSTGRES_MAXIMUM_CONNECTIONS` | `100` | Cluster connection capacity |
| `POSTGRES_CERTIFICATE_PATH` / `POSTGRES_PRIVATE_KEY_PATH` | unset | Explicit native TLS certificate/key pair |
| `POSTGRES_INITIALIZATION_TIMEOUT` | `2m` | New-cluster initialization deadline |
| `POSTGRES_STARTUP_TIMEOUT` | `60s` | Authenticated readiness deadline |
| `POSTGRES_SHUTDOWN_TIMEOUT` | `45s` | Fast-shutdown grace |
| `REDIS_ENABLED` | `true` | Enable managed Redis |
| `REDIS_BINARY` | `/usr/local/bin/sqldash-redis-server` | Isolated bundled-runtime launcher |
| `REDIS_MODULES_DIRECTORY` | `/opt/redis-runtime/modules` | Required bundled native modules |
| `REDIS_GATEWAY_ADDRESS` | `:6379` | Native listener |
| `REDIS_GATEWAY_ENABLED` | `true` | Run native Redis policy/TLS ingress |
| `REDIS_PUBLIC_HOST` | unset | Advertised native hostname |
| `REDIS_PUBLIC_PORT` | `6379` | Advertised external port |
| `REDIS_ALLOW_PLAINTEXT` | `true` | Explicitly allow plaintext Redis as well as TLS; never TLS fallback |
| `REDIS_CERTIFICATE_PATH` / `REDIS_PRIVATE_KEY_PATH` | unset | Explicit native Redis TLS certificate/key pair |
| `REDIS_MEMORY_LIMIT` | `268435456` (256 MiB) | Default per-database maxmemory |
| `REDIS_MEMORY_BUDGET` | `1073741824` (1 GiB) | Aggregate admission budget, including per-instance reserve |
| `REDIS_MAXIMUM_INSTANCES` | `16` | Instance ceiling; memory budget also applies |
| `REDIS_MAXIMUM_CONNECTIONS` | `100` | Native gateway total/per-database session ceiling; private clients have a separate reserve |
| `REDIS_MAXIMUM_VALUE_BYTES` | `8388608` (8 MiB) | Private-client bulk-value bound; Data previews have smaller caps |
| `REDIS_MAXIMUM_SCAN_BATCH` | `1000` | SCAN request ceiling; panel/provider limits can be smaller |
| `REDIS_MAXIMUM_STREAM_BUFFER` | `1000` | Bounded live-event queue length |
| `REDIS_STARTUP_TIMEOUT` | `60s` | Authenticated instance startup deadline |
| `REDIS_SHUTDOWN_TIMEOUT` | `45s` | Per-instance graceful-stop deadline |
| `REDIS_COMMAND_TIMEOUT` | `30s` | Private-client command timeout |
| `REDIS_REVALIDATION_INTERVAL` | `2s` | Native policy revalidation interval |
| `SESSION_COOKIE` | `sqldash` | Session cookie name |
| `ALLOWED_ORIGINS` | unset | Additional accepted origins (comma-separated) |
| `AUTOMATIC_CERTIFICATES` | `false` | Enable automatic certificate issuance |
| `CERTIFICATE_STAGING` | `false` | Use the staging certificate authority |
| `CERTIFICATE_EMAIL` | unset | Certificate issuer email |
| `CLOUDFLARE_API_TOKEN` | unset | Cloudflare API token for DNS-01 challenges |
| `HOSTING_ADDRESS` / `HOSTING_PASSWORD` / `HOSTING_APP` | unset | libSQL HTTP hostname registration/withdrawal |
