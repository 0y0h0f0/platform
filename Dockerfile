# Multi-stage Dockerfile for all services
# Build: docker build --build-arg SERVICE=api-gateway -t task-platform/api-gateway:latest .
#        docker build --build-arg SERVICE=user-service -t task-platform/user-service:latest .
#        docker build --build-arg SERVICE=task-service -t task-platform/task-service:latest .

FROM golang:1.26-alpine AS builder

ARG SERVICE=api-gateway

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/service ./cmd/${SERVICE}

FROM alpine:3.21

ARG SERVICE=api-gateway

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/service /app
COPY --from=builder /src/configs/docker/ /configs/docker/
COPY --from=builder /src/configs/security/ /configs/security/

ENV APP_ENV=docker
ENV CONFIG_FILE=/configs/docker/${SERVICE}.yaml

EXPOSE 8080 8081 8082 9091 9092

# Admin port differs per service; override via --build-arg (api-gateway:8080, user-svc:8081, task-svc:8082)
ARG ADMIN_PORT=8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
	CMD wget --no-verbose --tries=1 --spider http://localhost:${ADMIN_PORT}/healthz || exit 1

# Run as non-root
RUN adduser -D -g '' appuser
USER appuser

ENTRYPOINT ["/app"]
