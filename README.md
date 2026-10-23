# sqldash

A self-hosted, single-operator control panel for libSQL and PostgreSQL. Create databases, manage their connections, browse and edit data, run SQL, import and export backups, and inspect query activity.

Choose an engine when creating a database. One libSQL server hosts the libSQL databases, and one PostgreSQL 18 cluster hosts the PostgreSQL databases. Both run alongside the dashboard in the container. Existing databases retain their engine; there is no automatic conversion between libSQL and PostgreSQL.

## Run the container

```sh
docker build -t sqldash .
docker run --name sqldash \
  -p 80:80 -p 443:443 -p 5432:5432 \
  -e SQLDASH_DOMAIN=databases.example.com \
  -v sqldash-data:/data \
  --stop-timeout 70 \
  sqldash
```

Open the dashboard and create the operator account. That account owns the installation. SMTP is optional; recovery links can be copied when mail is not configured.

Keep `/data` persistent. Run **one container per data volume**, and stop the old container before starting its replacement. Never run overlapping PostgreSQL writers against the same volume.

The image includes PostgreSQL 18 server/client tools and libSQL. Its healthcheck verifies dashboard state, backend readiness and native ingress. PostgreSQL runs as a dedicated non-root OS user; the supervisor prepares only its own storage subtree.

## Database connections

### libSQL

Use the database's HTTP URL and an application token. The HTTP/Hrana proxy supports libSQL clients and WebSockets. Database hostnames select their namespace; clients can also use the supported namespace header when a database hostname is not supplied.

Configure database hostnames in DNS and your reverse proxy. Optional automatic certificates and hosting integration are described below.

### PostgreSQL

Creating a PostgreSQL database automatically creates its application login. The username is exactly the database name, and the password is randomly generated. Overview shows the username, current password and connection URI with masked secret fields, reveal, copy and rotation controls. The Connections tab shows current database sessions with scoped activity controls. Database passwords are stored encrypted and are separate from the dashboard account password.

The URI includes the password and omits the default port 5432. A configured non-default port remains explicit:

```text
postgresql://catalogue:<generated-password>@databases.example.com/catalogue?channel_binding=disable&sslmode=verify-full
```

Keep the URI private. Rotating the password invalidates the old password and closes existing application connections. Deleting the database removes its PostgreSQL login and stored password.

For an installation-managed certificate authority, download its public certificate through the trusted HTTPS dashboard and configure the client:

```sh
psql "host=databases.example.com dbname=catalogue user=catalogue sslmode=verify-full sslrootcert=./sqldash-postgres-ca.crt"
```

This command prompts for the generated database password. Supply the password through a protected password file for unattended clients, rather than putting it in shell history.

Native PostgreSQL needs a **raw TCP port**, normally 5432. An HTTP reverse proxy or a CapRover HTTPS hostname does not provide PostgreSQL ingress. The private postmaster listener, normally `127.0.0.1:55432`, must not be published.

The gateway authenticates through PostgreSQL using SCRAM, enforces current account/database/credential/network policy, and closes revoked sessions. PostgreSQL privileges enforce the credential's data profile. Native users never receive cluster-superuser or internal management credentials.

TLS is enabled and selected by default. Overview's Use SSL choice generates either a verified-TLS URI or an explicit `sslmode=disable` URI. The gateway supports both encrypted and explicitly unencrypted client connections; disabling SSL does not bypass authentication, permissions, network rules or session revocation. Unencrypted connections expose credentials and data to the network, so prefer verified TLS.

Certificate selection is: explicitly supplied native certificate/key, automatic certificate issuer configuration, or a persistent installation-managed CA. A configured certificate failure does not fall back to another trust root or plaintext, and a failed TLS handshake is never retried without TLS. Managed server certificates use ECDSA P-256; existing managed certificates are renewed under the same CA. Keep the CA private key persistent, and distribute only its public certificate/fingerprint.

Standard SCRAM over verified TLS is supported. The initial gateway topology does not support `channel_binding=require`/SCRAM-SHA-256-PLUS. PostgreSQL selects a database from the connection parameters, not the hostname alone. Databases share cluster resources, and ordinary PostgreSQL catalogs can reveal database/role names; this is not separate cluster isolation.

## Permissions

libSQL and PostgreSQL have separate permissions for database management, data, SQL, credentials, network rules and monitoring. The operator owns the installation and has installation-wide authority.

Existing account, role and sharing records are preserved. For any restricted account already configured, database visibility does not bypass engine-specific operation permissions. Existing libSQL grants retain libSQL access when upgrading; they do not automatically grant PostgreSQL access.

## Data and SQL

The explorer supports browsing, filtering, sorting and editing. PostgreSQL has schema navigation, composite primary keys and typed values, including precise numeric values, JSON, arrays, UUIDs and binary data. NULL, empty values and DEFAULT are distinct. Generated columns are not ordinary editable fields.

Rows without a stable key can be browsed and inserted but cannot be individually edited or deleted in the grid. Views and materialized views are not treated as ordinary editable tables.

The PostgreSQL console supports multiple result sets and cancellation. Transactions belong to one submission; unfinished transactions are rolled back before the connection is released. Structure controls manage database objects using scoped permissions, not a superuser console.

## Dumps, restores and cloning

- **libSQL:** SQLite files and SQLite SQL dumps; current and supported historical forks.
- **PostgreSQL:** native SQL dumps and custom archives, exported from a consistent snapshot. Restore and clone create a new database; they do not overwrite an existing database or convert another engine.
- **Duplicate:** use the database list's three-dot menu to copy its current contents into a new database of the same engine.
- **Remote PostgreSQL:** clone from an explicitly supplied host, port, database and credentials. The source is read without changing its data or configuration; target ownership and access are established locally. Verified TLS is the default; choose the source's trust mode or explicitly select No SSL for a server without TLS. There is no automatic downgrade. A failed submission keeps the remote-clone form open with its current inputs; passwords are not persisted by the form.

PostgreSQL restores use matching native tools and a target-only non-superuser login. Archive restoration does not adopt external database owners, login roles, grants or destinations. Plain SQL uploads support constrained dump input, not arbitrary psql programs: client shell commands, connection switching, file/program access and cluster-privilege operations are rejected.

Imports/clones do not inherit source application credentials or sharing. The current new-database import/clone workflow does not support extension-dependent dumps: it cannot enable extensions before the target exists, and it does not offer restore-in-place into an already prepared database. Such requirements are rejected without publishing a partially restored target. Extensions can be enabled afterward for ordinary database use through management.

Historical PostgreSQL/PITR restoration, replication management and major-version conversion are not automatic features. Logical exports are database backups, not complete installation backups.

## Extensions and monitoring

The PostgreSQL Extensions tab lists supported extensions actually present in the installation and their enablement state, with search by name and description. New databases enable `pg_stat_statements` automatically, including databases created by import or cloning. Other optional extensions start disabled and can be enabled or disabled only from a ready database's management screen, after creation. Enable required dependencies first. Disabling refuses to cascade into dependent objects.

Extension artifacts ship in the image, including pinned packages from the official PostgreSQL PGDG repository. The dashboard does not download extension code on a button click. Missing or unsupported extensions and unconfigured startup modules are omitted.

PostgreSQL query statistics use `pg_stat_statements`, enabled by default for new databases. Existing databases can enable it in Extensions; explicitly disabling it remains effective across restarts. Size and connection information remain available without it. Native statistics are aggregate counters: they do not provide libSQL's measured request percentiles, bytes, failure counts or rows-read counters. Unavailable metrics are labelled instead of displayed as zero. Query/activity data is restricted to authorized engines and databases.

Diagnostic and background-worker extensions are exposed only when their required runtime configuration is supported. Enabling an extension does not grant application logins cluster-superuser or unrestricted host-file access. Foreign data wrappers are not supported.

## Settings

All settings use the `SQLDASH_` prefix. The process reads environment variables and a `.env` in its **working directory**. Environment variables take precedence.

| Setting | Default | Purpose |
| --- | --- | --- |
| `DOMAIN` | `localhost` | Database hostname base and default native host |
| `PUBLIC_URL` | unset | Public dashboard origin for invitation/recovery links |
| `WEB_ADDRESS` | `127.0.0.1:7070` | Dashboard listener |
| `HTTP_PORT` / `HTTPS_PORT` | `8800` / `8443` | HTTP proxy ports; image uses 80/443 |
| `DATA_DIRECTORY` | `./data` | Persistent root; image uses `/data` |
| `SQLD_MANAGED` | `true` | Supervise libSQL |
| `SQLD_BINARY` | `sqld` | libSQL executable |
| `SQLD_ADDRESS` | `127.0.0.1:8080` | Private libSQL endpoint |
| `SQLD_ADMIN_ADDRESS` | `127.0.0.1:8081` | Private libSQL administration |
| `POSTGRES_ENABLED` | `true` | Make PostgreSQL available |
| `POSTGRES_MANAGED` | `true` | Supervise the dedicated PostgreSQL cluster |
| `POSTGRES_BINARY_DIRECTORY` | `/usr/lib/postgresql/18/bin` | Server and native client tools |
| `POSTGRES_ADDRESS` | `127.0.0.1:55432` | Private PostgreSQL endpoint |
| `POSTGRES_GATEWAY_ENABLED` | `true` | Run native policy/TLS ingress |
| `POSTGRES_GATEWAY_ADDRESS` | `:5432` | Native listener |
| `POSTGRES_PUBLIC_HOST` | domain | Advertised native hostname |
| `POSTGRES_PUBLIC_PORT` | `5432` | Advertised external port |
| `POSTGRES_ADMIN_USERNAME` | `sqldash_admin` | Private management principal |
| `POSTGRES_MAXIMUM_CONNECTIONS` | `100` | Cluster connection capacity |
| `POSTGRES_CERTIFICATE_PATH` / `POSTGRES_PRIVATE_KEY_PATH` | unset | Explicit native TLS certificate/key |
| `POSTGRES_INITIALIZATION_TIMEOUT` | `2m` | New-cluster initialization deadline |
| `POSTGRES_STARTUP_TIMEOUT` | `60s` | Authenticated readiness deadline |
| `POSTGRES_SHUTDOWN_TIMEOUT` | `45s` | Fast-shutdown grace |
| `SESSION_COOKIE` | `sqldash` | Session cookie name |
| `MAXIMUM_IMAGE_UPLOAD_BYTES` | `10485760` | Profile-picture limit |
| `ALLOWED_ORIGINS` | unset | Additional accepted origins |
| `DEBUG` | `false` | Development diagnostics |

Both engines are enabled by default. Explicitly disabling an engine is an installation setting, not a way to convert its existing databases. Managed PostgreSQL targets Linux containers; unsupported host platforms report an error rather than silently disabling PostgreSQL.

**Automatic certificates:** set `AUTOMATIC_CERTIFICATES=true`, `CLOUDFLARE_API_TOKEN` with zone DNS edit rights and `CERTIFICATE_EMAIL`. DNS-01 proves the requested names. `CERTIFICATE_STAGING=true` uses the staging authority. Native certificates have separate storage/trust handling from HTTP certificates.

**Hosting integration:** `HOSTING_ADDRESS`, `HOSTING_PASSWORD` and `HOSTING_APP` allow libSQL HTTP hostnames to be registered/withdrawn automatically. These settings do not publish PostgreSQL TCP ports.

### CapRover

Preserve the `/data` volume mapping. Keep one app instance and use stop-first replacement with sufficient stop grace. Map the native TCP port separately; use source-preserving ingress and verify the gateway sees the original client address before relying on client CIDRs. Do not automatically broaden rules for a NAT/proxy peer or trust HTTP forwarding headers as native client addresses.

CapRover's HTTP certificates are not automatically available inside the app. Configure native TLS or distribute the installation CA through the trusted dashboard.

## Development

```sh
make dev
```

Development runs the Linux image through `compose.dev.yml`, including on Windows. The dashboard is at `http://localhost:8800` (also `http://127.0.0.1:7070`), and native PostgreSQL uses `localhost:5432`. Both engines remain enabled; native Windows execution does not supervise PostgreSQL.

The development runner uses the existing `libsql-local` backend on host ports 8080/8081 without moving its databases. Panel state and PostgreSQL files persist in the `sqldash-development` Docker volume. Workspace templates and static assets are mounted read-only; rerun `make dev` to rebuild Go changes. `make dev-stop` stops the runner without deleting its volume.

When switching an existing host development installation, stop its processes and preserve the panel database, SQLite WAL, keys and other state before seeding the initially empty development volume. Never overwrite an existing volume or run the host and Docker panel against divergent writable copies. The original host files remain a backup, not an automatically synchronized second instance. Do not point development at production data.

`make setup` enables Git hooks; `make hooks` enables them separately. `make format` formats Django templates and CSS; `make format-check` checks them. Hooks preserve partial staging and validate formatting, syntax, build and vet. They do not run permanent repository tests.
