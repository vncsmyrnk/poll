default:
  @just list

run-migration name:
  go run ./cmd/migrations/main.go {{name}}

run-server:
  go run cmd/server/main.go

build-and-run-api-image:
  docker build --target api -t poll-api .
  docker compose up -d
  docker run -it --rm \
    --network host \
    -p 8080:8080 \
    --env-file ./.env \
    poll-api

build-and-run-summarizer-image:
  docker build --target votesummarizing -t poll-vote-summarizing .
  docker compose up -d
  docker run -it --rm \
    --network host \
    --env-file ./.env \
    poll-vote-summarizing

test:
  go test -v ./test/integration/...
