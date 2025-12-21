default:
  @just list

run-migration name:
  go run ./cmd/migrations/main.go {{name}}

run-server:
  go run cmd/server/main.go

run-vote-summary-generator:
  go run cmd/votesummarygenerator/main.go

run-sql-db file:
  #!/usr/bin/env bash
  source .env
  PGPASSWORD="$POSTGRES_PASSWORD" psql -U "$POSTGRES_USER" \
    -h "$POSTGRES_HOST" -d "$POSTGRES_DB" -p "$POSTGRES_PORT" -f {{file}}

build-and-run-api-image:
  docker build --target api -t poll-api .
  docker compose up -d
  docker run -it --rm \
    --network host \
    -p 8080:8080 \
    --env-file ./.env \
    poll-api

build-and-run-summarizer-image:
  docker build --target vote-summary-generator -t poll-vote-summary-generator .
  docker compose up -d
  docker run -it --rm \
    --network host \
    --env-file ./.env \
    poll-vote-summary-generator

test:
  go test -v ./test/integration/...
