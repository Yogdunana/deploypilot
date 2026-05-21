package gap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type TestGenerator struct {
	projectRoot string
	templates   *template.Template
}

type TestTemplate struct {
	PackageName   string
	Imports       []string
	FunctionName  string
	TargetFile    string
	TestCases     []TestCase
	SetupCode     string
	TeardownCode  string
}

type TestCase struct {
	Name        string
	Description string
	Input       string
	Output      string
	ShouldError bool
}

var testFileTemplate = template.Must(template.New("testfile").Parse(`package {{.PackageName}}

import (
	"context"
	"testing"
{{range .Imports}}	"{{.}}"
{{end}}
)

{{if .SetupCode}}{{.SetupCode}}{{end}}

func Test{{.FunctionName}}(t *testing.T) {
	tests := []struct {
		name        string
		description string
		input       interface{}
		want        interface{}
		wantErr     bool
	}{
{{range .TestCases}}		{
			name:        "{{.Name}}",
			description: "{{.Description}}",
			input:       {{.Input}},
			want:        {{.Output}},
			wantErr:     {{.ShouldError}},
		},
{{end}}	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			{{if .TargetFile}}target := New{{.FunctionName}}()
			got, err := target.{{.FunctionName}}(ctx{{if eq .TargetFile "service"}}, tt.input.({{.FunctionName}}Input){{end}})
			if (err != nil) != tt.wantErr {
				t.Errorf("{{.FunctionName}}() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("{{.FunctionName}}() = %v, want %v", got, tt.want)
			}
		{{end}}})
	}
}

{{if .TeardownCode}}{{.TeardownCode}}{{end}}
`))

var tableDrivenTemplate = template.Must(template.New("tabledriven").Parse(`package {{.PackageName}}

import (
	"testing"
{{range .Imports}}	"{{.}}"
{{end}}
)

func Test{{.FunctionName}}(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() interface{}
		run        func(interface{}) interface{}
		assert     func(interface{}, interface{}) bool
		wantErr    bool
	}{
{{range .TestCases}}		{
			name: "{{.Name}}",
			setup: func() interface{} {
				{{.Input}}
			},
			run: func(input interface{}) interface{} {
				{{if .TargetFile}}return {{.TargetFile}}(input){{end}}
			},
			assert: func(input, got interface{}) bool {
				{{.Output}}
				return true
			},
			wantErr: {{.ShouldError}},
		},
{{end}}	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.setup()
			got := tt.run(input)
			if !tt.assert(input, got) {
				t.Errorf("test %s failed", tt.name)
			}
		})
	}
}
`))

var concurrencyTestTemplate = template.Must(template.New("concurrency").Parse(`package {{.PackageName}}

import (
	"context"
	"sync"
	"testing"
	"time"
{{range .Imports}}	"{{.}}"
{{end}}
)

func Test{{.FunctionName}}Concurrency(t *testing.T) {
	const goroutines = 100
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	errCh := make(chan error, goroutines)
{{if .TargetFile}}target := New{{.FunctionName}}()
{{end}}	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := target.{{.FunctionName}}(ctx{{if eq .TargetFile "service"}}, generateTestInput(id){{end}})
			if err != nil {
				errCh <- fmt.Errorf("goroutine %d: %w", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrency error: %v", err)
	}
}

func Test{{.FunctionName}}RaceCondition(t *testing.T) {
{{if .TargetFile}}target := New{{.FunctionName}}()
{{end}}	var mu sync.Mutex
	counter := 0

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			counter++
		}()
	}
	wg.Wait()

	if counter != 1000 {
		t.Errorf("race condition detected: expected 1000, got %d", counter)
	}
}
`))

var handlerTestTemplate = template.Must(template.New("handler").Parse(`package {{.PackageName}}

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
{{range .Imports}}	"{{.}}"
{{end}}
)

func Test{{.FunctionName}}Handler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		setupRouter    func(*httptest.Server) *ServeMux
	}{
{{range .TestCases}}		{
			name:           "{{.Name}}",
			method:         "GET",
			path:           "/{{.FunctionName}}",
			body:           nil,
			expectedStatus: http.StatusOK,
		},
{{end}}	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("expected method %s, got %s", tt.method, r.Method)
				}
				if r.URL.Path != tt.path {
					t.Errorf("expected path %s, got %s", tt.path, r.URL.Path)
				}
				w.WriteHeader(tt.expectedStatus)
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			}))
			defer server.Close()

			resp, err := http.Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func Test{{.FunctionName}}HandlerValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
{{range .TestCases}}		{
			name:        "{{.Name}}",
			input:       "{{.Input}}",
			expectError: {{.ShouldError}},
		},
{{end}}	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validate{{.FunctionName}}(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("validation error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
`))

func NewTestGenerator(projectRoot string) *TestGenerator {
	return &TestGenerator{
		projectRoot: projectRoot,
	}
}

func (g *TestGenerator) GenerateTestFile(targetFile string, functionName string, testType string) (string, error) {
	tmpl := g.selectTemplate(testType)
	if tmpl == nil {
		return "", fmt.Errorf("unknown test type: %s", testType)
	}

	pkgName := g.getPackageName(targetFile)
	imports := g.getRequiredImports(targetFile, testType)

	data := TestTemplate{
		PackageName:  pkgName,
		Imports:      imports,
		FunctionName: functionName,
		TargetFile:   targetFile,
		TestCases:    g.generateTestCases(functionName, testType),
		SetupCode:    g.generateSetupCode(functionName, testType),
		TeardownCode: g.generateTeardownCode(functionName, testType),
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return sb.String(), nil
}

func (g *TestGenerator) selectTemplate(testType string) *template.Template {
	switch testType {
	case "basic":
		return testFileTemplate
	case "table":
		return tableDrivenTemplate
	case "concurrency":
		return concurrencyTestTemplate
	case "handler":
		return handlerTestTemplate
	default:
		return testFileTemplate
	}
}

func (g *TestGenerator) getPackageName(filePath string) string {
	relPath, err := filepath.Rel(g.projectRoot, filePath)
	if err != nil {
		return "main"
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) >= 2 {
		return parts[1]
	}
	return "main"
}

func (g *TestGenerator) getRequiredImports(filePath string, testType string) []string {
	imports := []string{
		"context",
		"testing",
	}

	lower := strings.ToLower(filePath)

	if strings.Contains(lower, "model") {
		imports = append(imports, "\"github.com/Yogdunana/deploypilot/internal/model\"")
	}
	if strings.Contains(lower, "service") {
		imports = append(imports, "\"github.com/Yogdunana/deploypilot/internal/service\"")
	}
	if strings.Contains(lower, "mcp") {
		imports = append(imports, "\"github.com/Yogdunana/deploypilot/internal/mcp\"")
	}
	if strings.Contains(lower, "engine") {
		imports = append(imports, "\"github.com/Yogdunana/deploypilot/internal/engine/builder\"")
		imports = append(imports, "\"github.com/Yogdunana/deploypilot/internal/engine/deployer\"")
	}

	switch testType {
	case "table":
		imports = append(imports, "\"fmt\"")
	case "concurrency":
		imports = append(imports, "\"sync\"", "\"time\"", "\"fmt\"")
	case "handler":
		imports = append(imports, "\"net/http\"", "\"net/http/httptest\"", "\"encoding/json\"")
	}

	return imports
}

func (g *TestGenerator) generateTestCases(functionName, testType string) []TestCase {
	cases := []TestCase{
		{
			Name:        "valid_input",
			Description: "should handle valid input correctly",
			Input:       "validInput()",
			Output:      "expectedOutput",
			ShouldError: false,
		},
		{
			Name:        "empty_input",
			Description: "should handle empty input gracefully",
			Input:       "emptyInput()",
			Output:      "nil",
			ShouldError: true,
		},
		{
			Name:        "invalid_input",
			Description: "should reject invalid input",
			Input:       "invalidInput()",
			Output:      "nil",
			ShouldError: true,
		},
	}

	if testType == "concurrency" {
		cases = append(cases, TestCase{
			Name:        "concurrent_access",
			Description: "should handle concurrent access safely",
			Input:       "concurrentInput()",
			Output:      "expectedOutput",
			ShouldError: false,
		})
	}

	return cases
}

func (g *TestGenerator) generateSetupCode(functionName, testType string) string {
	switch testType {
	case "handler":
		return `
func setupTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}))
}
`
	case "concurrency":
		return `
type mock{{.FunctionName}} struct{}

func (m *mock{{.FunctionName}}) {{.FunctionName}}(ctx context.Context{{if eq .TargetFile "service"}}, input interface{}{{end}}) (interface{}, error) {
	return "result", nil
}
`
	default:
		return ""
	}
}

func (g *TestGenerator) generateTeardownCode(functionName, testType string) string {
	return ""
}

func (g *TestGenerator) WriteTestFile(targetFile, functionName, testType string) (string, error) {
	content, err := g.GenerateTestFile(targetFile, functionName, testType)
	if err != nil {
		return "", err
	}

	ext := filepath.Ext(targetFile)
	dir := filepath.Dir(targetFile)
	baseName := filepath.Base(targetFile)
	funcName := strings.TrimSuffix(baseName, ext)

	testFile := filepath.Join(dir, funcName+"_test.go")

	if _, err := os.Stat(testFile); err == nil {
		return "", fmt.Errorf("test file already exists: %s", testFile)
	}

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write test file: %w", err)
	}

	return testFile, nil
}

func (g *TestGenerator) GenerateForGap(gap *CoverageGap) ([]string, error) {
	files := make([]string, 0)

	testType := g.inferTestType(gap.FilePath)

	baseName := filepath.Base(gap.FilePath)
	funcName := strings.TrimSuffix(baseName, ".go")

	testFile, err := g.WriteTestFile(gap.FilePath, funcName, testType)
	if err != nil {
		return files, err
	}
	files = append(files, testFile)

	return files, nil
}

func (g *TestGenerator) inferTestType(filePath string) string {
	lower := strings.ToLower(filePath)

	if strings.Contains(lower, "handler") || strings.Contains(lower, "api") {
		return "handler"
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "basic"
	}

	if strings.Contains(string(content), "goroutine") ||
		strings.Contains(string(content), "sync.") ||
		strings.Contains(string(content), "channel") {
		return "concurrency"
	}

	return "basic"
}

func (g *TestGenerator) GenerateReport(analysis *GapAnalysisResult) string {
	var sb strings.Builder

	sb.WriteString("# Test Gap Analysis Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", analysis.Timestamp.Format("2006-01-02 15:04:05")))

	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- Total Files Analyzed: %d\n", analysis.TotalFilesAnalyzed))
	sb.WriteString(fmt.Sprintf("- Average Coverage: %.1f%%\n", analysis.Summary.AvgCoverage))
	sb.WriteString(fmt.Sprintf("- Total Gaps Found: %d\n\n", analysis.Summary.TotalGaps))

	sb.WriteString("### Risk Distribution\n\n")
	sb.WriteString(fmt.Sprintf("- High Risk: %d\n", analysis.Summary.HighPriorityCount))
	sb.WriteString(fmt.Sprintf("- Medium Risk: %d\n", analysis.Summary.MediumPriorityCount))
	sb.WriteString(fmt.Sprintf("- Low Risk: %d\n\n", analysis.Summary.LowPriorityCount))

	if len(analysis.Summary.CriticalModules) > 0 {
		sb.WriteString("### Critical Modules Requiring Attention\n\n")
		for _, module := range analysis.Summary.CriticalModules {
			sb.WriteString(fmt.Sprintf("- %s\n", module))
		}
		sb.WriteString("\n")
	}

	if len(analysis.HighRiskGaps) > 0 {
		sb.WriteString("## High Risk Gaps\n\n")
		for _, gap := range analysis.HighRiskGaps {
			sb.WriteString(fmt.Sprintf("### %s\n", filepath.Base(gap.FilePath)))
			sb.WriteString(fmt.Sprintf("- File: `%s`\n", gap.FilePath))
			sb.WriteString(fmt.Sprintf("- Risk Level: %s\n", gap.RiskLevel))
			sb.WriteString(fmt.Sprintf("- Risk Score: %.1f\n", gap.RiskScore))
			sb.WriteString(fmt.Sprintf("- Reason: %s\n", gap.Reason))
			if len(gap.RiskFactors) > 0 {
				sb.WriteString("- Risk Factors:\n")
				for _, factor := range gap.RiskFactors {
					sb.WriteString(fmt.Sprintf("  - %s\n", factor))
				}
			}
			if len(gap.RecommendedTests) > 0 {
				sb.WriteString("- Recommended Tests:\n")
				for _, test := range gap.RecommendedTests {
					sb.WriteString(fmt.Sprintf("  - `%s`\n", test))
				}
			}
			sb.WriteString("\n")
		}
	}

	if len(analysis.MediumRiskGaps) > 0 {
		sb.WriteString("## Medium Risk Gaps\n\n")
		for _, gap := range analysis.MediumRiskGaps[:5] {
			sb.WriteString(fmt.Sprintf("- `%s` - %s\n", filepath.Base(gap.FilePath), gap.Reason))
		}
		if len(analysis.MediumRiskGaps) > 5 {
			sb.WriteString(fmt.Sprintf("- ... and %d more\n", len(analysis.MediumRiskGaps)-5))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
