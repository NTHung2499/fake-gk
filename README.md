# FakeGK

A small ChatGPT-style web chatbot built with Go, Gin, MySQL, and server-rendered HTML.

## Run Locally

```bash
APP_SECRET=dev-only-fakegk-secret-change-me go run ./cmd/fake-gk
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
APP_SECRET=dev-only-fakegk-secret-change-me
OPENAI_MODEL=gpt-5.4-mini
OPENAI_FAST_MODEL=gpt-5.4-mini
OPENAI_DEEP_MODEL=gpt-5.5
OPENAI_ROUTER_MODEL=gpt-5.4-mini
CHAT_CONTEXT_MESSAGES=30
OPENAI_REQUEST_TIMEOUT_SECONDS=60
```

`APP_SECRET` is used to encrypt user-provided OpenAI API keys before storing them in MySQL. Use a strong secret in shared or deployed environments.

## Data Model

The app auto-creates these tables on startup:

```text
users
user_api_keys
chat_sessions
chat_messages
```

Users are anonymous browser users identified by an HTTP-only cookie. Each user can store one encrypted OpenAI API key and create multiple chat sessions with persisted message history.

## Docker

```bash
docker build -t fake-gk .
docker run --rm -p 3000:3000 -e APP_SECRET=change-this-secret fake-gk
```

## Health Checks

```text
GET /healthz
GET /readyz
```

`/readyz` checks the MySQL connection and returns `503` when the database is unavailable.
