default:
  @just list

run-migration name:
  go run ./cmd/migrations/main.go {{name}}

run-server:
  go run cmd/server/main.go

test:
  go test -v ./test/integration/...
