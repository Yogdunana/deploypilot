# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM --platform=$BUILDPLATFORM golang:1.23.6 AS backend

# Install xx for cross-compilation support
COPY --from=tonistiigi/xx / /

# Install native GCC and cross-compilation toolchain for arm64
RUN apt-get update && apt-get install -y --no-install-recommends gcc && rm -rf /var/lib/apt/lists/*
RUN xx-apt install -y --no-install-recommends gcc-aarch64-linux-gnu libc6-dev-arm64-cross && xx-apt clean

RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist

# Generate swagger docs (excluded by .dockerignore)
RUN swag init -g cmd/api-server/main.go -o docs/swagger

ARG TARGETPLATFORM
ENV CGO_ENABLED=1

# Use xx-go to handle cross-compilation automatically
RUN xx-go --wrap && \
    go build -ldflags="-s -w" -o /deploypilot ./cmd/deploypilot/ && \
    go build -ldflags="-s -w" -o /api-server ./cmd/api-server/ && \
    go build -ldflags="-s -w" -o /mcp-server ./cmd/mcp-server/ && \
    xx-verify /deploypilot && xx-verify /api-server && xx-verify /mcp-server

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata openssh-client
WORKDIR /app
COPY --from=backend /deploypilot .
COPY --from=backend /api-server .
COPY --from=backend /mcp-server .
EXPOSE 8080
CMD ["./api-server"]
