default:
  @just list

run-migration name:
  go run ./cmd/migrations/main.go {{name}}
