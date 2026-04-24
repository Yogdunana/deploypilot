# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /mcp-server ./cmd/mcp-server/
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /deploypilot ./cmd/deploypilot/

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates openssh-client

WORKDIR /app

COPY --from=builder /mcp-server .
COPY --from=builder /deploypilot .
COPY config.yaml .

EXPOSE 8080

ENTRYPOINT ["/mcp-server"]
CMD ["--config", "config.yaml"]
