# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend (multi-arch with CGO)
FROM --platform=$BUILDPLATFORM golang:1.23 AS backend
RUN apt-get update && apt-get install -y --no-install-recommends gcc-aarch64-linux-gnu gcc-x86-64-linux-gnu && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist

ARG TARGETOS
ARG TARGETARCH

# Set the correct CC based on target architecture
RUN if [ "$TARGETARCH" = "arm64" ]; then \
      export CC=aarch64-linux-gnu-gcc && \
      export CGO_ENABLED=1 && \
      go build -ldflags="-s -w" -o /deploypilot ./cmd/deploypilot/ && \
      go build -ldflags="-s -w" -o /api-server ./cmd/api-server/ && \
      go build -ldflags="-s -w" -o /mcp-server ./cmd/mcp-server/; \
    else \
      export CC=gcc && \
      export CGO_ENABLED=1 && \
      go build -ldflags="-s -w" -o /deploypilot ./cmd/deploypilot/ && \
      go build -ldflags="-s -w" -o /api-server ./cmd/api-server/ && \
      go build -ldflags="-s -w" -o /mcp-server ./cmd/mcp-server/; \
    fi

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata openssh-client
WORKDIR /app
COPY --from=backend /deploypilot .
COPY --from=backend /api-server .
COPY --from=backend /mcp-server .
EXPOSE 8080
CMD ["./api-server"]
