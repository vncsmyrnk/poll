# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API
RUN go build -o /app/bin/api ./cmd/server/main.go

# Build Vote Summarizing
RUN go build -o /app/bin/votesummarizing ./cmd/votesummarizing/main.go

# API Image
FROM alpine:latest AS api
RUN addgroup -S nonroot && adduser -S nonroot -G nonroot
WORKDIR /app
COPY --from=builder /app/bin/api .
USER nonroot
CMD ["./api"]

# Worker Image
FROM alpine:latest AS votesummarizing
RUN addgroup -S nonroot && adduser -S nonroot -G nonroot
WORKDIR /app
COPY --from=builder /app/bin/votesummarizing .
USER nonroot
CMD ["./votesummarizing"]
