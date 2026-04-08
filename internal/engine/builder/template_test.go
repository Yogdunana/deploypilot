package builder

import (
	"strings"
	"testing"
)

func TestAllTemplatesRegistered(t *testing.T) {
	r := NewRegistry()
	templates := r.List()

	if len(templates) != 9 {
		t.Errorf("Expected 9 templates, got %d", len(templates))
	}
}

func TestTemplateNode(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplateNode)

	if tmpl == nil {
		t.Fatal("Node template not found")
	}
	if tmpl.Name != "Node.js" {
		t.Errorf("Name = %q", tmpl.Name)
	}
	if tmpl.Port != 3000 {
		t.Errorf("Port = %d, want 3000", tmpl.Port)
	}
	if tmpl.HealthPath != "/health" {
		t.Errorf("HealthPath = %q", tmpl.HealthPath)
	}
	if tmpl.Image != "node:18-alpine" {
		t.Errorf("Image = %q", tmpl.Image)
	}
	if !strings.Contains(tmpl.Dockerfile, "node:18-alpine") {
		t.Error("Dockerfile should contain node:18-alpine")
	}
	if !strings.Contains(tmpl.Dockerfile, "EXPOSE 3000") {
		t.Error("Dockerfile should EXPOSE 3000")
	}
	if tmpl.EnvVars["NODE_ENV"] != "production" {
		t.Errorf("NODE_ENV = %q", tmpl.EnvVars["NODE_ENV"])
	}
}

func TestTemplatePython(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplatePython)

	if tmpl == nil {
		t.Fatal("Python template not found")
	}
	if tmpl.Port != 8000 {
		t.Errorf("Port = %d, want 8000", tmpl.Port)
	}
	if !strings.Contains(tmpl.Dockerfile, "python:3.11") {
		t.Error("Dockerfile should contain python:3.11")
	}
	if !strings.Contains(tmpl.Dockerfile, "gunicorn") {
		t.Error("Dockerfile should contain gunicorn")
	}
}

func TestTemplateGo(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplateGo)

	if tmpl == nil {
		t.Fatal("Go template not found")
	}
	if tmpl.Port != 8080 {
		t.Errorf("Port = %d, want 8080", tmpl.Port)
	}
	if tmpl.HealthPath != "/healthz" {
		t.Errorf("HealthPath = %q, want /healthz", tmpl.HealthPath)
	}
	if !strings.Contains(tmpl.Dockerfile, "golang:1.22") {
		t.Error("Dockerfile should contain golang:1.22")
	}
	if !strings.Contains(tmpl.Dockerfile, "CGO_ENABLED=0") {
		t.Error("Dockerfile should use CGO_ENABLED=0 for static binary")
	}
	if !strings.Contains(tmpl.Dockerfile, "multi-stage") {
		// Check for multi-stage pattern
		if strings.Count(tmpl.Dockerfile, "FROM") < 2 {
			t.Error("Go Dockerfile should be multi-stage")
		}
	}
}

func TestTemplateJava(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplateJava)

	if tmpl == nil {
		t.Fatal("Java template not found")
	}
	if !strings.Contains(tmpl.Dockerfile, "eclipse-temurin:17") {
		t.Error("Dockerfile should contain eclipse-temurin:17")
	}
	if !strings.Contains(tmpl.Dockerfile, "mvn") {
		t.Error("Dockerfile should contain mvn")
	}
	if !strings.Contains(tmpl.Dockerfile, "EXPOSE 8080") {
		t.Error("Dockerfile should EXPOSE 8080")
	}
}

func TestTemplatePHP(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplatePHP)

	if tmpl == nil {
		t.Fatal("PHP template not found")
	}
	if tmpl.Port != 9000 {
		t.Errorf("Port = %d, want 9000", tmpl.Port)
	}
	if !strings.Contains(tmpl.Dockerfile, "composer") {
		t.Error("Dockerfile should contain composer")
	}
}

func TestTemplateRuby(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplateRuby)

	if tmpl == nil {
		t.Fatal("Ruby template not found")
	}
	if !strings.Contains(tmpl.Dockerfile, "ruby:3.3") {
		t.Error("Dockerfile should contain ruby:3.3")
	}
	if !strings.Contains(tmpl.Dockerfile, "bundle") {
		t.Error("Dockerfile should contain bundle")
	}
	if tmpl.EnvVars["RAILS_ENV"] != "production" {
		t.Errorf("RAILS_ENV = %q", tmpl.EnvVars["RAILS_ENV"])
	}
}

func TestTemplateRust(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplateRust)

	if tmpl == nil {
		t.Fatal("Rust template not found")
	}
	if !strings.Contains(tmpl.Dockerfile, "rust:1.77") {
		t.Error("Dockerfile should contain rust:1.77")
	}
	if !strings.Contains(tmpl.Dockerfile, "cargo build --release") {
		t.Error("Dockerfile should contain cargo build --release")
	}
}

func TestTemplateStatic(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplateStatic)

	if tmpl == nil {
		t.Fatal("Static template not found")
	}
	if tmpl.Port != 80 {
		t.Errorf("Port = %d, want 80", tmpl.Port)
	}
	if !strings.Contains(tmpl.Dockerfile, "nginx") {
		t.Error("Dockerfile should contain nginx")
	}
}

func TestTemplateDockerCustom(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplateDocker)

	if tmpl == nil {
		t.Fatal("Docker template not found")
	}
	if tmpl.Image != "" {
		t.Errorf("Custom Docker template should have empty Image, got %q", tmpl.Image)
	}
	if tmpl.Dockerfile != "" {
		t.Error("Custom Docker template should have empty Dockerfile")
	}
}

func TestFindByType(t *testing.T) {
	r := NewRegistry()

	tmpl, err := r.FindByType("node")
	if err != nil {
		t.Fatalf("FindByType(node) error = %v", err)
	}
	if tmpl.Type != TemplateNode {
		t.Errorf("Type = %q, want node", tmpl.Type)
	}

	// Case insensitive
	tmpl2, err := r.FindByType("Python")
	if err != nil {
		t.Fatalf("FindByType(Python) error = %v", err)
	}
	if tmpl2.Type != TemplatePython {
		t.Errorf("Type = %q, want python", tmpl2.Type)
	}
}

func TestFindByTypeUnknown(t *testing.T) {
	r := NewRegistry()

	_, err := r.FindByType("unknown")
	if err == nil {
		t.Error("FindByType(unknown) should fail")
	}
	if !strings.Contains(err.Error(), "unknown template type") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestGetUnknown(t *testing.T) {
	r := NewRegistry()

	tmpl := r.Get(TemplateType("nonexistent"))
	if tmpl != nil {
		t.Error("Get(unknown) should return nil")
	}
}

func TestGenerateDockerfile(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplateNode)

	df := tmpl.GenerateDockerfile(map[string]string{})
	if df == "" {
		t.Error("GenerateDockerfile should not be empty")
	}
	if !strings.Contains(df, "FROM") {
		t.Error("Dockerfile should contain FROM")
	}
}

func TestGenerateDockerfileCustomTemplate(t *testing.T) {
	r := NewRegistry()
	tmpl := r.Get(TemplateDocker)

	df := tmpl.GenerateDockerfile(nil)
	if !strings.Contains(df, "provide your own Dockerfile") {
		t.Errorf("Custom template Dockerfile = %q", df)
	}
}

func TestHealthCheckURL(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		tmplType TemplateType
		host     string
		want     string
	}{
		{TemplateNode, "example.com", "http://example.com:3000/health"},
		{TemplatePython, "example.com", "http://example.com:8000/health"},
		{TemplateGo, "example.com", "http://example.com:8080/healthz"},
		{TemplateJava, "example.com", "http://example.com:8080/actuator/health"},
		{TemplateStatic, "example.com", "http://example.com:80/"},
	}

	for _, tc := range tests {
		tmpl := r.Get(tc.tmplType)
		url := tmpl.HealthCheckURL(tc.host)
		if url != tc.want {
			t.Errorf("HealthCheckURL(%s) = %q, want %q", tc.tmplType, url, tc.want)
		}
	}
}

func TestAllTemplatesHaveRequiredFields(t *testing.T) {
	r := NewRegistry()

	for _, tmpl := range r.List() {
		t.Run(string(tmpl.Type), func(t *testing.T) {
			if tmpl.Type == "" {
				t.Error("Type should not be empty")
			}
			if tmpl.Name == "" {
				t.Error("Name should not be empty")
			}
			if tmpl.Description == "" {
				t.Error("Description should not be empty")
			}
			if tmpl.Port <= 0 {
				t.Errorf("Port = %d, should be > 0", tmpl.Port)
			}
		})
	}
}

func TestHealthCheckURL_EmptyPath(t *testing.T) {
	tmpl := AppTemplate{HealthPath: "", Port: 8080}
	url := tmpl.HealthCheckURL("localhost")
	if url != "" {
		t.Errorf("expected empty URL for empty HealthPath, got %q", url)
	}
}
