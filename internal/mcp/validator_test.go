package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- ValidateParams tests ---

func TestValidateParams_MissingRequiredField(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name parameter"),
		),
		mcp.WithString("optional_field",
			mcp.Description("Optional parameter"),
		),
	)

	args := map[string]interface{}{}
	err := ValidateParams("test_tool", tool, args)

	if err == nil {
		t.Fatal("expected validation error for missing required field")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if ve.ToolName != "test_tool" {
		t.Errorf("ToolName = %q, want %q", ve.ToolName, "test_tool")
	}

	if len(ve.Fields) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(ve.Fields))
	}

	if ve.Fields[0].Field != "name" {
		t.Errorf("Field = %q, want %q", ve.Fields[0].Field, "name")
	}

	if ve.Fields[0].Reason != "required field is missing" {
		t.Errorf("Reason = %q, want %q", ve.Fields[0].Reason, "required field is missing")
	}
}

func TestValidateParams_MultipleMissingRequiredFields(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("field_a", mcp.Required(), mcp.Description("Field A")),
		mcp.WithString("field_b", mcp.Required(), mcp.Description("Field B")),
		mcp.WithString("field_c", mcp.Required(), mcp.Description("Field C")),
	)

	args := map[string]interface{}{}
	err := ValidateParams("test_tool", tool, args)

	if err == nil {
		t.Fatal("expected validation error")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if len(ve.Fields) != 3 {
		t.Fatalf("expected 3 field errors, got %d", len(ve.Fields))
	}
}

func TestValidateParams_WrongType(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name parameter"),
		),
	)

	// Pass an integer where a string is expected
	args := map[string]interface{}{
		"name": 12345,
	}
	err := ValidateParams("test_tool", tool, args)

	if err == nil {
		t.Fatal("expected validation error for wrong type")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if len(ve.Fields) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(ve.Fields))
	}

	if !strings.Contains(ve.Fields[0].Reason, "expected type string") {
		t.Errorf("Reason = %q, want to contain 'expected type string'", ve.Fields[0].Reason)
	}
}

func TestValidateParams_WrongTypeNumber(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithNumber("count",
			mcp.Required(),
			mcp.Description("Count parameter"),
		),
	)

	args := map[string]interface{}{
		"count": "not_a_number",
	}
	err := ValidateParams("test_tool", tool, args)

	if err == nil {
		t.Fatal("expected validation error for wrong type")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if !strings.Contains(ve.Fields[0].Reason, "expected type number") {
		t.Errorf("Reason = %q, want to contain 'expected type number'", ve.Fields[0].Reason)
	}
}

func TestValidateParams_WrongTypeBoolean(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithBoolean("enabled",
			mcp.Required(),
			mcp.Description("Enabled parameter"),
		),
	)

	args := map[string]interface{}{
		"enabled": "true",
	}
	err := ValidateParams("test_tool", tool, args)

	if err == nil {
		t.Fatal("expected validation error for wrong type")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if !strings.Contains(ve.Fields[0].Reason, "expected type boolean") {
		t.Errorf("Reason = %q, want to contain 'expected type boolean'", ve.Fields[0].Reason)
	}
}

func TestValidateParams_WrongTypeArray(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithArray("items",
			mcp.Required(),
			mcp.Description("Items parameter"),
		),
	)

	args := map[string]interface{}{
		"items": "not_an_array",
	}
	err := ValidateParams("test_tool", tool, args)

	if err == nil {
		t.Fatal("expected validation error for wrong type")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if !strings.Contains(ve.Fields[0].Reason, "expected type array") {
		t.Errorf("Reason = %q, want to contain 'expected type array'", ve.Fields[0].Reason)
	}
}

func TestValidateParams_WrongTypeObject(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithObject("config",
			mcp.Required(),
			mcp.Description("Config parameter"),
		),
	)

	args := map[string]interface{}{
		"config": "not_an_object",
	}
	err := ValidateParams("test_tool", tool, args)

	if err == nil {
		t.Fatal("expected validation error for wrong type")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if !strings.Contains(ve.Fields[0].Reason, "expected type object") {
		t.Errorf("Reason = %q, want to contain 'expected type object'", ve.Fields[0].Reason)
	}
}

func TestValidateParams_InvalidEnumValue(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("status",
			mcp.Required(),
			mcp.Description("Status parameter"),
			mcp.Enum("active", "inactive", "pending"),
		),
	)

	args := map[string]interface{}{
		"status": "unknown",
	}
	err := ValidateParams("test_tool", tool, args)

	if err == nil {
		t.Fatal("expected validation error for invalid enum value")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if !strings.Contains(ve.Fields[0].Reason, "not one of the allowed values") {
		t.Errorf("Reason = %q, want to contain 'not one of the allowed values'", ve.Fields[0].Reason)
	}
}

func TestValidateParams_ValidEnumValue(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("status",
			mcp.Required(),
			mcp.Description("Status parameter"),
			mcp.Enum("active", "inactive", "pending"),
		),
	)

	args := map[string]interface{}{
		"status": "active",
	}
	err := ValidateParams("test_tool", tool, args)

	if err != nil {
		t.Fatalf("expected no error for valid enum value, got: %v", err)
	}
}

func TestValidateParams_ValidParams(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name parameter"),
		),
		mcp.WithString("optional_field",
			mcp.Description("Optional parameter"),
		),
	)

	args := map[string]interface{}{
		"name":           "test-app",
		"optional_field": "optional-value",
	}
	err := ValidateParams("test_tool", tool, args)

	if err != nil {
		t.Fatalf("expected no error for valid params, got: %v", err)
	}
}

func TestValidateParams_NoInputSchema(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A tool with no params"),
	)

	args := map[string]interface{}{}
	err := ValidateParams("test_tool", tool, args)

	if err != nil {
		t.Fatalf("expected no error for tool with no inputSchema, got: %v", err)
	}
}

func TestValidateParams_NilArgs(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name parameter"),
		),
	)

	err := ValidateParams("test_tool", tool, nil)

	if err == nil {
		t.Fatal("expected validation error for nil args with required field")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if len(ve.Fields) != 1 || ve.Fields[0].Field != "name" {
		t.Errorf("expected missing 'name' field error, got: %v", ve.Fields)
	}
}

func TestValidateParams_ExtraFieldsIgnored(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name parameter"),
		),
	)

	args := map[string]interface{}{
		"name":        "test-app",
		"extra_field": "should be ignored",
	}
	err := ValidateParams("test_tool", tool, args)

	if err != nil {
		t.Fatalf("expected no error for valid params with extra fields, got: %v", err)
	}
}

func TestValidateParams_NumberTypeAcceptsFloat64(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithNumber("score",
			mcp.Required(),
			mcp.Description("Score parameter"),
		),
	)

	args := map[string]interface{}{
		"score": 3.14,
	}
	err := ValidateParams("test_tool", tool, args)

	if err != nil {
		t.Fatalf("expected no error for float64 number, got: %v", err)
	}
}

func TestValidateParams_NumberTypeAcceptsInt(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithNumber("count",
			mcp.Required(),
			mcp.Description("Count parameter"),
		),
	)

	args := map[string]interface{}{
		"count": 42,
	}
	err := ValidateParams("test_tool", tool, args)

	if err != nil {
		t.Fatalf("expected no error for int number, got: %v", err)
	}
}

// --- ValidationError.Error() tests ---

func TestValidationError_ErrorFormat(t *testing.T) {
	ve := &ValidationError{
		ToolName: "deploy_app",
		Fields: []FieldError{
			{Field: "image", Reason: "required field is missing"},
			{Field: "container_name", Reason: "required field is missing"},
		},
	}

	msg := ve.Error()
	if !strings.Contains(msg, "[deploy_app] INVALID_PARAMS") {
		t.Errorf("Error() = %q, want to contain '[deploy_app] INVALID_PARAMS'", msg)
	}
	if !strings.Contains(msg, "image: required field is missing") {
		t.Errorf("Error() = %q, want to contain 'image: required field is missing'", msg)
	}
	if !strings.Contains(msg, "container_name: required field is missing") {
		t.Errorf("Error() = %q, want to contain 'container_name: required field is missing'", msg)
	}
}

// --- InvalidParamsResult tests ---

func TestInvalidParamsResult_Format(t *testing.T) {
	fields := []FieldError{
		{Field: "image", Reason: "required field is missing", Example: "nginx:latest"},
		{Field: "container_name", Reason: "required field is missing"},
	}

	result := InvalidParamsResult("deploy_app", fields)

	if !result.IsError {
		t.Error("expected IsError to be true")
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	// Parse the content
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
		t.Fatalf("failed to parse content as JSON: %v", err)
	}

	if parsed["code"] != "INVALID_PARAMS" {
		t.Errorf("code = %v, want INVALID_PARAMS", parsed["code"])
	}

	msg, _ := parsed["message"].(string)
	if !strings.Contains(msg, "deploy_app") {
		t.Errorf("message = %q, want to contain 'deploy_app'", msg)
	}

	details, ok := parsed["details"].(map[string]interface{})
	if !ok {
		t.Fatal("expected details to be a map")
	}

	imageDetail, ok := details["image"].(map[string]interface{})
	if !ok {
		t.Fatal("expected image detail to be a map")
	}
	if imageDetail["reason"] != "required field is missing" {
		t.Errorf("image reason = %v, want 'required field is missing'", imageDetail["reason"])
	}
	if imageDetail["example"] != "nginx:latest" {
		t.Errorf("image example = %v, want 'nginx:latest'", imageDetail["example"])
	}
}

func TestInvalidParamsResult_NoExample(t *testing.T) {
	fields := []FieldError{
		{Field: "field1", Reason: "required field is missing"},
	}

	result := InvalidParamsResult("test_tool", fields)

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
		t.Fatalf("failed to parse content as JSON: %v", err)
	}

	details := parsed["details"].(map[string]interface{})
	fieldDetail := details["field1"].(map[string]interface{})

	// example should not be present when empty
	if _, exists := fieldDetail["example"]; exists {
		t.Error("expected no 'example' field when empty")
	}
}

// --- withValidation wrapper tests ---

func TestWithValidation_PassesOnValidParams(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name parameter"),
		),
	)

	handlerCalled := false
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlerCalled = true
		return mcp.NewToolResultText("ok"), nil
	}

	wrapped := withValidation("test_tool", tool, handler)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "test_tool",
			Arguments: map[string]interface{}{"name": "test"},
		},
	}

	result, err := wrapped(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("expected handler to be called for valid params")
	}
	if result.IsError {
		t.Errorf("expected success result, got error: %v", result.Content)
	}
}

func TestWithValidation_ReturnsErrorOnInvalidParams(t *testing.T) {
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name parameter"),
		),
	)

	handlerCalled := false
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlerCalled = true
		return mcp.NewToolResultText("ok"), nil
	}

	wrapped := withValidation("test_tool", tool, handler)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "test_tool",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := wrapped(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handlerCalled {
		t.Error("expected handler NOT to be called for invalid params")
	}
	if !result.IsError {
		t.Error("expected error result for invalid params")
	}

	// Verify structured error content
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
		t.Fatalf("failed to parse content as JSON: %v", err)
	}

	if parsed["code"] != "INVALID_PARAMS" {
		t.Errorf("code = %v, want INVALID_PARAMS", parsed["code"])
	}
}

// --- Helper function tests ---

func TestGetExample(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected string
	}{
		{
			name:     "explicit example",
			schema:   map[string]interface{}{"description": "some desc", "example": "nginx:latest"},
			expected: "nginx:latest",
		},
		{
			name:     "default value",
			schema:   map[string]interface{}{"description": "some desc", "default": "main"},
			expected: "main",
		},
		{
			name:     "example in description with e.g.",
			schema:   map[string]interface{}{"description": "Docker image (e.g. nginx:latest)"},
			expected: "nginx:latest",
		},
		{
			name:     "example in description with default:",
			schema:   map[string]interface{}{"description": "Git branch (default: main)"},
			expected: "main",
		},
		{
			name:     "no example available",
			schema:   map[string]interface{}{"description": "Just a description"},
			expected: "",
		},
		{
			name:     "nil schema",
			schema:   nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.schema != nil {
				result = getExample(tt.schema)
			} else {
				result = getExample(nil)
			}
			if result != tt.expected {
				t.Errorf("getExample() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsNumber(t *testing.T) {
	tests := []struct {
		value  interface{}
		expect bool
	}{
		{float64(3.14), true},
		{int(42), true},
		{int64(100), true},
		{"not a number", false},
		{true, false},
		{nil, false},
	}

	for _, tt := range tests {
		if got := isNumber(tt.value); got != tt.expect {
			t.Errorf("isNumber(%v) = %v, want %v", tt.value, got, tt.expect)
		}
	}
}

func TestIsInteger(t *testing.T) {
	tests := []struct {
		value  interface{}
		expect bool
	}{
		{int(42), true},
		{int64(100), true},
		{float64(3.0), true},
		{float64(3.14), false},
		{"not an int", false},
		{true, false},
	}

	for _, tt := range tests {
		if got := isInteger(tt.value); got != tt.expect {
			t.Errorf("isInteger(%v) = %v, want %v", tt.value, got, tt.expect)
		}
	}
}

// --- Integration test: withValidation matches server.ToolHandlerFunc ---

func TestWithValidation_IsToolHandlerFunc(t *testing.T) {
	// Verify that withValidation returns a server.ToolHandlerFunc
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("Test"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
	)

	var handler server.ToolHandlerFunc = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	}

	wrapped := withValidation("test_tool", tool, handler)

	// Compile-time check: wrapped should be a server.ToolHandlerFunc
	var _ server.ToolHandlerFunc = wrapped

	_ = wrapped
}
