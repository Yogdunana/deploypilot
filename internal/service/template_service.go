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
