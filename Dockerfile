# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.23-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /deploypilot ./cmd/deploypilot/
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /api-server ./cmd/api-server/
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /mcp-server ./cmd/mcp-server/

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata openssh-client
WORKDIR /app
COPY --from=backend /deploypilot .
COPY --from=backend /api-server .
COPY --from=backend /mcp-server .
EXPOSE 8080
CMD ["./api-server"]
