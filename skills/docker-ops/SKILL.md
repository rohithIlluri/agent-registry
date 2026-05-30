---
name: docker-ops
description: "Docker and Docker Compose operations: build, run, inspect logs, manage containers and images."
version: "1.1.0"
license: MIT
keywords: [docker, containers, compose, devops, build]
category: devops
allowed-tools: [Bash, Read]
user-invocable: true
---

# Docker Operations

Assist with Docker and Docker Compose operations.

## When to invoke

Invoke for any Docker or container-related task: building images, running containers, debugging, cleanup.

## Common operations

### Build
```bash
docker build -t <image>:<tag> .
docker buildx build --platform linux/amd64,linux/arm64 -t <image>:<tag> --push .
```

### Run / inspect
```bash
docker run --rm -it <image> <cmd>
docker logs -f <container>
docker exec -it <container> sh
docker inspect <container>
```

### Compose
```bash
docker compose up -d
docker compose logs -f <service>
docker compose down --volumes
docker compose ps
```

### Cleanup (use with caution)
```bash
docker system prune -f          # remove stopped containers, unused networks, dangling images
docker image prune -a -f        # remove ALL unused images
docker volume prune -f          # remove unused volumes
```

## Dockerfile best practices

- Use specific base image tags, never `latest`
- Multi-stage builds for smaller production images
- `.dockerignore` to exclude `node_modules`, `.git`, build artifacts
- Non-root user: `RUN adduser -D app && USER app`
- `COPY --chown=app:app` to set ownership at copy time
- Layer order: system deps → app deps → source code (for cache efficiency)

## Constraints

- Always confirm before running `docker system prune` or `--volumes`
- Do not expose secrets in `docker run -e` — suggest `.env` files or secrets management
