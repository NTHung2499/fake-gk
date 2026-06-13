# Fake GK

A small Google Keep-like notes app built with Node.js, Express, EJS, and MySQL.

## Run Locally

```bash
cp .env.example .env
npm install
npm start
```

Default environment values are ready for the Kubernetes MySQL lab:

```text
PORT=3000
MYSQL_HOST=mysql.database.svc.cluster.local
MYSQL_PORT=3306
MYSQL_DATABASE=appdb
MYSQL_USER=appuser
MYSQL_PASSWORD=apppass123
```

The app auto-creates the `notes` table on startup.

Commit `package-lock.json` after running `npm install`; the Docker build will use `npm ci` when the lockfile exists.

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
