# Poll API

A high-performance, scalable polling system built with Go. Designed for flexibility, it runs seamlessly in on-premise Docker environments and scales effortlessly for cloud deployments.

Its purpose is to provide a WEB REST API for creating and listing polls, and also enable users to vote on their options. It currently provides unauthenticated voting with IP restriction.

## 📦 Installation

This project provides pre-built Docker images and a `docker-compose` setup for easy on-premise deployment. The REST API images are published at [dockerhub](https://hub.docker.com/repository/docker/vncsmyrnk/poll-api).

## ☁️ Cloud Environment

We offer a managed cloud environment for users who prefer a hosted solution.
Visit [poll.vncsmyrnk.dev](https://poll.vncsmyrnk.dev) to get started with the managed API.

## 🛠️ Development

To run this project locally, check the `justfile`.

### Running Tests

We use `testcontainers-go` for robust integration testing.

```bash
go test ./test/integration/... -v
```
