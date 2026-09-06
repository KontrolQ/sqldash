# sqldash

A control panel for libSQL, for one operator and their own databases. It runs the
databases, and gives you one place to browse them, write SQL against them, import and
export them, and see what they are doing.

It is a single binary that serves rendered pages. There is no API to call and nothing
is passed in a query string, so a database is only ever reached by someone signed in.

## Running it

```sh
make setup
make build
SQLDASH_DOMAIN=localhost ./bin/sqldash
```

The panel answers on `SQLDASH_WEB_ADDRESS`, `127.0.0.1:7070` by default. The first
account you create owns the instance.

The image is self-contained: it carries `sqld` alongside the binary, so a container
needs nothing but a data directory.

```sh
docker build -t sqldash .
docker run -p 80:80 -p 443:443 -e SQLDASH_DOMAIN=example.com -v sqldash:/data sqldash
```

## Settings

Every setting is an environment variable prefixed `SQLDASH_`, read from the environment
or from a `.env` beside the binary. Only the domain has to be set.

| Setting | Default | What it is |
| --- | --- | --- |
| `DOMAIN` | `localhost` | the name databases are addressed under |
| `WEB_ADDRESS` | `127.0.0.1:7070` | where the panel itself listens |
| `HTTP_PORT` / `HTTPS_PORT` | `8800` / `8443` | where the database proxy listens |
| `REDIRECT_TO_TLS` | `true` | send plain requests to HTTPS |
| `DATA_DIRECTORY` | `./data` | databases, imports, certificates and its own state |
| `SQLD_MANAGED` | `true` | run libSQL itself; set false to use one already running |
| `SQLD_BINARY` | `sqld` | the libSQL server to run when managing it |
| `SQLD_ADDRESS` | `127.0.0.1:8080` | where that server answers |
| `SQLD_ADMIN_ADDRESS` | `127.0.0.1:8081` | where its namespace administration answers |
| `SESSION_COOKIE` | `sqldash` | the name of the sign-in cookie |
| `ALLOWED_ORIGINS` | | extra origins to accept, comma separated |
| `DEBUG` | `false` | log every query and request |

## A hostname for each database

Databases are addressed by name: `catalogue.example.com` reaches the database called
`catalogue`. The proxy decides which database a request means from the hostname it
arrives on, so a client cannot ask for another one by sending a different header.

Two optional groups of settings support that.

**Certificates.** With `AUTOMATIC_CERTIFICATES=true`, a wildcard is issued for the
domain and kept renewed. A wildcard can only be proved over DNS-01, so this needs
`CLOUDFLARE_API_TOKEN` with DNS edit rights on the zone, and `CERTIFICATE_EMAIL` to
register with the authority. `CERTIFICATE_STAGING=true` issues against the staging
authority while you are working it out.

**Hosting.** If the panel runs behind something that terminates TLS per hostname,
`HOSTING_ADDRESS`, `HOSTING_PASSWORD` and `HOSTING_APP` let it register and withdraw
`<database>.<domain>` as databases are created and deleted. All three must be set for
any of it to happen; unset, none of this code runs and everything else works the same.

## Importing and exporting

A database can be exported as SQL and rebuilt from it. An import is replayed statement
by statement over the wire rather than handed to the server as a file path, so it works
against a server that cannot see your filesystem — a container, or one on another host.
