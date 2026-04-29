package service

import (
	"context"
	"fmt"
)

// ---------- 23. ListTemplates ----------

func (b *Bridge) ListTemplates(ctx context.Context) (interface{}, error) {
	templates := []map[string]interface{}{
		{"type": "node", "name": "Node.js", "description": "Express / Fastify / Next.js application", "build_cmd": "npm install && npm run build", "run_cmd": "node dist/main.js", "port": 3000},
		{"type": "python", "name": "Python", "description": "Flask / FastAPI / Django application", "build_cmd": "pip install -r requirements.txt", "run_cmd": "gunicorn app:app", "port": 8000},
		{"type": "go", "name": "Go", "description": "Go HTTP server or CLI tool", "build_cmd": "go build -o app .", "run_cmd": "./app", "port": 8080},
		{"type": "java", "name": "Java", "description": "Spring Boot / Quarkus application", "build_cmd": "./mvnw package -DskipTests", "run_cmd": "java -jar target/app.jar", "port": 8080},
		{"type": "php", "name": "PHP", "description": "Laravel / Symfony application", "build_cmd": "composer install", "run_cmd": "php artisan serve", "port": 8000},
		{"type": "ruby", "name": "Ruby", "description": "Rails / Sinatra application", "build_cmd": "bundle install", "run_cmd": "bundle exec rails server", "port": 3000},
		{"type": "rust", "name": "Rust", "description": "Actix / Axum / Warp application", "build_cmd": "cargo build --release", "run_cmd": "./target/release/app", "port": 8080},
		{"type": "static", "name": "Static Site", "description": "Nginx-hosted static HTML/CSS/JS", "build_cmd": "", "run_cmd": "nginx -g 'daemon off;'", "port": 80},
		{"type": "docker", "name": "Docker", "description": "Custom Docker image deployment", "build_cmd": "", "run_cmd": "", "port": 0},
	}
	return templates, nil
}

// ---------- 24. GetTemplate ----------

func (b *Bridge) GetTemplate(ctx context.Context, tmplType string) (interface{}, error) {
	all, _ := b.ListTemplates(ctx)
	tmpls := all.([]map[string]interface{})

	for _, t := range tmpls {
		if t["type"] == tmplType {
			return t, nil
		}
	}
	return nil, fmt.Errorf("template type %q not found; available: node, python, go, java, php, ruby, rust, static, docker", tmplType)
}

// ---------- 25. ListEnvTemplates ----------

func (b *Bridge) ListEnvTemplates(ctx context.Context) (interface{}, error) {
	templates := []map[string]interface{}{
		{
			"service_type":  "mysql",
			"display_name":  "MySQL",
			"description":   "MySQL relational database server",
			"env_vars": []map[string]interface{}{
				{"name": "MYSQL_ROOT_PASSWORD", "description": "Root user password for MySQL", "required": true, "default_value": "", "example": "secure_password_123"},
				{"name": "MYSQL_DATABASE", "description": "Name of the default database to create", "required": false, "default_value": "", "example": "myapp"},
				{"name": "MYSQL_USER", "description": "Additional user to create", "required": false, "default_value": "", "example": "appuser"},
				{"name": "MYSQL_PASSWORD", "description": "Password for the additional user", "required": false, "default_value": "", "example": "user_password"},
				{"name": "MYSQL_PORT", "description": "Port on which MySQL listens", "required": false, "default_value": "3306", "example": "3306"},
			},
		},
		{
			"service_type":  "redis",
			"display_name":  "Redis",
			"description":   "Redis in-memory key-value data store",
			"env_vars": []map[string]interface{}{
				{"name": "REDIS_PORT", "description": "Port on which Redis listens", "required": false, "default_value": "6379", "example": "6379"},
				{"name": "REDIS_PASSWORD", "description": "Password for Redis authentication", "required": false, "default_value": "", "example": "redis_password"},
				{"name": "REDIS_MAXMEMORY", "description": "Maximum memory Redis can use", "required": false, "default_value": "", "example": "256mb"},
				{"name": "REDIS_APPENDONLY", "description": "Enable AOF persistence (yes/no)", "required": false, "default_value": "yes", "example": "yes"},
			},
		},
		{
			"service_type":  "postgresql",
			"display_name":  "PostgreSQL",
			"description":   "PostgreSQL relational database server",
			"env_vars": []map[string]interface{}{
				{"name": "POSTGRES_DB", "description": "Name of the default database to create", "required": false, "default_value": "", "example": "myapp"},
				{"name": "POSTGRES_USER", "description": "Superuser name", "required": false, "default_value": "", "example": "appuser"},
				{"name": "POSTGRES_PASSWORD", "description": "Superuser password", "required": true, "default_value": "", "example": "secure_password_123"},
				{"name": "POSTGRES_PORT", "description": "Port on which PostgreSQL listens", "required": false, "default_value": "5432", "example": "5432"},
			},
		},
		{
			"service_type":  "mongodb",
			"display_name":  "MongoDB",
			"description":   "MongoDB document-oriented NoSQL database",
			"env_vars": []map[string]interface{}{
				{"name": "MONGO_INITDB_ROOT_USERNAME", "description": "Root username for MongoDB", "required": false, "default_value": "", "example": "root"},
				{"name": "MONGO_INITDB_ROOT_PASSWORD", "description": "Root password for MongoDB", "required": true, "default_value": "", "example": "secure_password_123"},
				{"name": "MONGO_INITDB_DATABASE", "description": "Name of the default database to create", "required": false, "default_value": "", "example": "myapp"},
				{"name": "MONGO_PORT", "description": "Port on which MongoDB listens", "required": false, "default_value": "27017", "example": "27017"},
			},
		},
		{
			"service_type":  "nginx",
			"display_name":  "Nginx",
			"description":   "Nginx HTTP and reverse proxy server",
			"env_vars": []map[string]interface{}{
				{"name": "NGINX_PORT", "description": "Port on which Nginx listens", "required": false, "default_value": "80", "example": "80"},
				{"name": "NGINX_SERVER_NAME", "description": "Server name / domain for the default virtual host", "required": false, "default_value": "", "example": "example.com"},
				{"name": "NGINX_SSL_CERT_PATH", "description": "Path to the SSL certificate file", "required": false, "default_value": "", "example": "/etc/nginx/ssl/cert.pem"},
				{"name": "NGINX_SSL_KEY_PATH", "description": "Path to the SSL private key file", "required": false, "default_value": "", "example": "/etc/nginx/ssl/key.pem"},
				{"name": "NGINX_WORKER_PROCESSES", "description": "Number of worker processes (auto or a number)", "required": false, "default_value": "auto", "example": "auto"},
				{"name": "NGINX_CLIENT_MAX_BODY_SIZE", "description": "Maximum allowed size of the client request body", "required": false, "default_value": "64m", "example": "64m"},
			},
		},
	}
	return templates, nil
}

// ---------- 26. GetEnvTemplate ----------

func (b *Bridge) GetEnvTemplate(ctx context.Context, serviceType string) (interface{}, error) {
	all, _ := b.ListEnvTemplates(ctx)
	tmpls := all.([]map[string]interface{})

	for _, t := range tmpls {
		if t["service_type"] == serviceType {
			return t, nil
		}
	}
	return nil, fmt.Errorf("env template for service type %q not found; available: mysql, redis, postgresql, mongodb, nginx", serviceType)
}
