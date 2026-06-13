# Fake GK

A small notes app built with Go, Gin, MySQL, and server-rendered HTML.

## Run Locally

```bash
cp .env.example .env
go run ./cmd/fake-gk
```

Default environment values are ready for the Kubernetes MySQL lab:

```text
PORT=3000
MYSQL_HOST=mysql.database.svc.cluster.local
MYSQL_PORT=3306
MYSQL_DATABASE=appdb
MYSQL_USER=appuser
MYSQL_PASSWORD=apppass123
MYSQL_CONNECTION_LIMIT=10
```

The app auto-creates the `notes` table on startup and reuses the same schema as the previous Node.js version.

## Docker

```bash
docker build -t fake-gk .
docker run --rm -p 3000:3000 fake-gk
```

## Health Checks

```text
GET /healthz
GET /readyz
```

`/readyz` checks the MySQL connection and returns `503` when the database is unavailable.
