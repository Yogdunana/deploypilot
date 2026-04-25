# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend (multi-arch with CGO)
FROM --platform=$BUILDPLATFORM tonistiigi/xx AS xx
FROM --platform=$BUILDPLATFORM golang:1.23 AS backend

# Install xx-go for CGO cross-compilation
COPY --from=xx / /

ARG TARGETPLATFORM
WORKDIR /app

# Install cross-compilation C toolchain
RUN apt-get update && apt-get install -y --no-install-recommends clang lld && rm -rf /var/lib/apt/lists/*
RUN xx-apt install -y gcc libc6-dev

COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist

# Enable CGO and build with xx-go
ENV CGO_ENABLED=1
RUN xx-go build -ldflags="-s -w" -o /deploypilot ./cmd/deploypilot/ && \
    xx-go build -ldflags="-s -w" -o /api-server ./cmd/api-server/ && \
    xx-go build -ldflags="-s -w" -o /mcp-server ./cmd/mcp-server/

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata openssh-client
WORKDIR /app
COPY --from=backend /deploypilot .
COPY --from=backend /api-server .
COPY --from=backend /mcp-server .
EXPOSE 8080
CMD ["./api-server"]
