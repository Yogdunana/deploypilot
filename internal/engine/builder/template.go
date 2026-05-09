package builder

import (
	"fmt"
	"strings"
)

// TemplateType identifies the type of application template.
type TemplateType string

const (
	TemplateNode     TemplateType = "node"
	TemplatePython   TemplateType = "python"
	TemplateGo       TemplateType = "go"
	TemplateJava     TemplateType = "java"
	TemplatePHP      TemplateType = "php"
	TemplateRuby     TemplateType = "ruby"
	TemplateRust     TemplateType = "rust"
	TemplateStatic   TemplateType = "static"
	TemplateDocker   TemplateType = "docker"
)

// AppTemplate defines a deployment template for a specific tech stack.
type AppTemplate struct {
	Type         TemplateType   `json:"type"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Image        string         `json:"image"`          // default Docker image
	Port         int            `json:"port"`           // default internal port
	HealthPath   string         `json:"health_path"`    // default health check path
	BuildCmd     string         `json:"build_cmd"`      // build command
	StartCmd     string         `json:"start_cmd"`      // start command
	EnvVars      map[string]string `json:"env_vars"`    // default environment variables
	Dockerfile   string         `json:"dockerfile"`     // Dockerfile template
}

// TemplateRegistry holds all available templates.
type TemplateRegistry struct {
	templates map[TemplateType]*AppTemplate
}

// NewRegistry creates a new TemplateRegistry with all built-in templates.
func NewRegistry() *TemplateRegistry {
	r := &TemplateRegistry{
		templates: make(map[TemplateType]*AppTemplate),
	}

	// Register all 9 templates
	r.register(&AppTemplate{
		Type:        TemplateNode,
		Name:        "Node.js",
		Description: "Node.js application with npm/pnpm",
		Image:       "node:18-alpine",
		Port:        3000,
		HealthPath:  "/health",
		BuildCmd:    "npm install && npm run build",
		StartCmd:    "node dist/server.js",
		EnvVars:     map[string]string{"NODE_ENV": "production"},
		Dockerfile: `FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --production
COPY . .
RUN npm run build

FROM node:18-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 3000
CMD ["node", "dist/server.js"]`,
	})

	r.register(&AppTemplate{
		Type:        TemplatePython,
		Name:        "Python",
		Description: "Python application with pip/venv",
		Image:       "python:3.11-slim",
		Port:        8000,
		HealthPath:  "/health",
		BuildCmd:    "pip install -r requirements.txt",
		StartCmd:    "gunicorn app:app --bind 0.0.0.0:8000",
		EnvVars:     map[string]string{"PYTHONUNBUFFERED": "1"},
		Dockerfile: `FROM python:3.11-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

FROM python:3.11-slim
WORKDIR /app
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY . .
EXPOSE 8000
CMD ["gunicorn", "app:app", "--bind", "0.0.0.0:8000"]`,
	})

	r.register(&AppTemplate{
		Type:        TemplateGo,
		Name:        "Go",
		Description: "Go application with multi-stage build",
		Image:       "golang:1.22-alpine",
		Port:        8080,
		HealthPath:  "/healthz",
		BuildCmd:    "go build -o app .",
		StartCmd:    "./app",
		EnvVars:     map[string]string{"GIN_MODE": "release"},
		Dockerfile: `FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o app .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/app .
EXPOSE 8080
CMD ["./app"]`,
	})

	r.register(&AppTemplate{
		Type:        TemplateJava,
		Name:        "Java (Spring Boot)",
		Description: "Java Spring Boot application with Maven",
		Image:       "eclipse-temurin:17-jdk-alpine",
		Port:        8080,
		HealthPath:  "/actuator/health",
		BuildCmd:    "mvn clean package -DskipTests",
		StartCmd:    "java -jar target/app.jar",
		EnvVars:     map[string]string{"JAVA_OPTS": "-Xmx512m"},
		Dockerfile: `FROM eclipse-temurin:17-jdk-alpine AS builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline -B
COPY src ./src
RUN mvn clean package -DskipTests -B

FROM eclipse-temurin:17-jre-alpine
WORKDIR /app
COPY --from=builder /app/target/*.jar app.jar
EXPOSE 8080
CMD ["java", "-jar", "app.jar"]`,
	})

	r.register(&AppTemplate{
		Type:        TemplatePHP,
		Name:        "PHP",
		Description: "PHP application with Composer",
		Image:       "php:8.3-fpm-alpine",
		Port:        9000,
		HealthPath:  "/health",
		BuildCmd:    "composer install --no-dev",
		StartCmd:    "php-fpm",
		EnvVars:     map[string]string{"APP_ENV": "production"},
		Dockerfile: `FROM composer:2 AS builder
WORKDIR /app
COPY composer.json composer.lock* ./
RUN composer install --no-dev --optimize-autoloader --no-interaction
COPY . .

FROM php:8.3-fpm-alpine
COPY --from=builder /app /app
WORKDIR /app
EXPOSE 9000
CMD ["php-fpm"]`,
	})

	r.register(&AppTemplate{
		Type:        TemplateRuby,
		Name:        "Ruby (Rails)",
		Description: "Ruby on Rails application",
		Image:       "ruby:3.3-alpine",
		Port:        3000,
		HealthPath:  "/up",
		BuildCmd:    "bundle install && bundle exec rails assets:precompile",
		StartCmd:    "bundle exec rails server -b 0.0.0.0",
		EnvVars:     map[string]string{"RAILS_ENV": "production", "RAILS_LOG_TO_STDOUT": "1"},
		Dockerfile: `FROM ruby:3.3-alpine AS builder
WORKDIR /app
RUN apk add --no-cache build-base libxml2-dev
COPY Gemfile Gemfile.lock ./
RUN bundle install --deployment --without development test
COPY . .
RUN bundle exec rails assets:precompile

FROM ruby:3.3-alpine
WORKDIR /app
COPY --from=builder /app /app
EXPOSE 3000
CMD ["bundle", "exec", "rails", "server", "-b", "0.0.0.0"]`,
	})

	r.register(&AppTemplate{
		Type:        TemplateRust,
		Name:        "Rust",
		Description: "Rust application with cargo",
		Image:       "rust:1.77-alpine",
		Port:        8080,
		HealthPath:  "/health",
		BuildCmd:    "cargo build --release",
		StartCmd:    "./app",
		EnvVars:     map[string]string{"RUST_LOG": "info"},
		Dockerfile: `FROM rust:1.77-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev
COPY Cargo.toml Cargo.lock* ./
RUN mkdir src && echo "fn main() {}" > src/main.rs && cargo build --release && rm -rf src
COPY src ./src
RUN touch src/main.rs && cargo build --release

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/target/release/app .
EXPOSE 8080
CMD ["./app"]`,
	})

	r.register(&AppTemplate{
		Type:        TemplateStatic,
		Name:        "Static Site",
		Description: "Static HTML/CSS/JS site with Nginx",
		Image:       "nginx:alpine",
		Port:        80,
		HealthPath:  "/",
		BuildCmd:    "",
		StartCmd:    "nginx -g 'daemon off;'",
		EnvVars:     map[string]string{},
		Dockerfile: `FROM nginx:alpine
COPY . /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]`,
	})

	r.register(&AppTemplate{
		Type:        TemplateDocker,
		Name:        "Custom Docker",
		Description: "Custom Dockerfile (user-provided)",
		Image:       "",
		Port:        8080,
		HealthPath:  "/health",
		BuildCmd:    "",
		StartCmd:    "",
		EnvVars:     map[string]string{},
		Dockerfile:  "", // user provides their own Dockerfile
	})

	return r
}

func (r *TemplateRegistry) register(t *AppTemplate) {
	r.templates[t.Type] = t
}

// Get returns a template by type, or nil if not found.
func (r *TemplateRegistry) Get(t TemplateType) *AppTemplate {
	return r.templates[t]
}

// List returns all registered templates.
func (r *TemplateRegistry) List() []*AppTemplate {
	var list []*AppTemplate
	for _, t := range r.templates {
		list = append(list, t)
	}
	return list
}

// FindByType finds a template by string type name (case-insensitive).
func (r *TemplateRegistry) FindByType(name string) (*AppTemplate, error) {
	t := TemplateType(strings.ToLower(name))
	if tmpl, ok := r.templates[t]; ok {
		return tmpl, nil
	}
	return nil, fmt.Errorf("unknown template type: %s (available: %s)", name, r.availableTypes())
}

func (r *TemplateRegistry) availableTypes() string {
	var types []string
	for t := range r.templates {
		types = append(types, string(t))
	}
	return strings.Join(types, ", ")
}

// GenerateDockerfile generates a Dockerfile from a template with custom overrides.
func (t *AppTemplate) GenerateDockerfile(overrides map[string]string) string {
	dockerfile := t.Dockerfile
	if dockerfile == "" {
		return "# Custom Dockerfile - provide your own Dockerfile in the project root"
	}
	for key, value := range overrides {
		dockerfile = strings.ReplaceAll(dockerfile, "{{"+key+"}}", value)
	}
	return dockerfile
}

// HealthCheckURL returns the full health check URL for a given host.
func (t *AppTemplate) HealthCheckURL(host string) string {
	if t.HealthPath == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d%s", host, t.Port, t.HealthPath)
}
