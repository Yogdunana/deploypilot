package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FieldError describes a single parameter validation error.
type FieldError struct {
	Field   string `json:"field"`
	Reason  string `json:"reason"`
	Example string `json:"example,omitempty"`
}

// ValidationError represents structured parameter validation errors for a tool.
type ValidationError struct {
	ToolName string       `json:"tool_name"`
	Fields   []FieldError `json:"fields"`
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = fmt.Sprintf("%s: %s", f.Field, f.Reason)
	}
	return fmt.Sprintf("[%s] INVALID_PARAMS: %s", e.ToolName, strings.Join(parts, "; "))
}

// ValidateParams validates tool arguments against the tool's inputSchema.
// Returns a structured ValidationError if validation fails, nil if all params are valid.
func ValidateParams(toolName string, tool mcp.Tool, args map[string]interface{}) error {
	schema := tool.InputSchema
	if schema.Properties == nil {
		return nil
	}

	var errors []FieldError

	// Check required fields
	for _, reqField := range schema.Required {
		if _, exists := args[reqField]; !exists {
			propSchema := schema.Properties[reqField]
			errors = append(errors, FieldError{
				Field:   reqField,
				Reason:  "required field is missing",
				Example: getExample(propSchema),
			})
		}
	}

	// Check type constraints for provided fields
	for field, value := range args {
		propSchema, exists := schema.Properties[field]
		if !exists {
			continue
		}
		if ferr := validateType(field, propSchema, value); ferr != nil {
			errors = append(errors, *ferr)
		}
	}

	if len(errors) > 0 {
		return &ValidationError{ToolName: toolName, Fields: errors}
	}
	return nil
}

// validateType checks if the provided value matches the expected JSON Schema type.
func validateType(field string, propSchema, value interface{}) *FieldError {
	if propSchema == nil {
		return nil
	}

	// propSchema is map[string]interface{} from ToolInputSchema.Properties
	schemaMap, ok := propSchema.(map[string]interface{})
	if !ok {
		return nil
	}

	schemaType, _ := schemaMap["type"].(string)
	if schemaType == "" {
		return nil
	}

	// Check enum constraint first
	if ferr := validateEnum(field, schemaMap, value); ferr != nil {
		return ferr
	}

	switch schemaType {
	case "string":
		if _, ok := value.(string); !ok {
			return &FieldError{
				Field:   field,
				Reason:  fmt.Sprintf("expected type string, got %T", value),
				Example: getExample(propSchema),
			}
		}
	case "number":
		if !isNumber(value) {
			return &FieldError{
				Field:   field,
				Reason:  fmt.Sprintf("expected type number, got %T", value),
				Example: getExample(propSchema),
			}
		}
	case "integer":
		if !isInteger(value) {
			return &FieldError{
				Field:   field,
				Reason:  fmt.Sprintf("expected type integer, got %T", value),
				Example: getExample(propSchema),
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return &FieldError{
				Field:   field,
				Reason:  fmt.Sprintf("expected type boolean, got %T", value),
				Example: getExample(propSchema),
			}
		}
	case "array":
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return &FieldError{
				Field:   field,
				Reason:  fmt.Sprintf("expected type array, got %T", value),
				Example: getExample(propSchema),
			}
		}
	case "object":
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Map {
			return &FieldError{
				Field:   field,
				Reason:  fmt.Sprintf("expected type object, got %T", value),
				Example: getExample(propSchema),
			}
		}
	}

	return nil
}

// validateEnum checks if the provided value is one of the allowed enum values.
func validateEnum(field string, schemaMap map[string]interface{}, value interface{}) *FieldError {
	enumRaw, exists := schemaMap["enum"]
	if !exists {
		return nil
	}

	enumSlice, ok := enumRaw.([]interface{})
	if !ok {
		return nil
	}

	for _, allowed := range enumSlice {
		if value == allowed {
			return nil
		}
		// Handle numeric comparison (JSON numbers are float64)
		if vf, ok := value.(float64); ok {
			if af, ok := allowed.(float64); ok && vf == af {
				return nil
			}
		}
	}

	// Build a human-readable list of allowed values
	allowedStrs := make([]string, len(enumSlice))
	for i, v := range enumSlice {
		allowedStrs[i] = fmt.Sprintf("%v", v)
	}

	return &FieldError{
		Field:   field,
		Reason:  fmt.Sprintf("value %v is not one of the allowed values: %s", value, strings.Join(allowedStrs, ", ")),
		Example: allowedStrs[0],
	}
}

// getExample extracts an example value from a property schema.
// It tries the "example" field first, then falls back to the description.
func getExample(propSchema interface{}) string {
	if propSchema == nil {
		return ""
	}

	schemaMap, ok := propSchema.(map[string]interface{})
	if !ok {
		return ""
	}

	// Try explicit "example" field
	if ex, ok := schemaMap["example"]; ok {
		return fmt.Sprintf("%v", ex)
	}

	// Try "default" field
	if def, ok := schemaMap["default"]; ok {
		return fmt.Sprintf("%v", def)
	}

	// Try extracting from description
	if desc, ok := schemaMap["description"].(string); ok {
		// Look for patterns like "(e.g. xxx)" or "(default: xxx)"
		examples := extractExampleFromDescription(desc)
		if examples != "" {
			return examples
		}
	}

	return ""
}

// extractExampleFromDescription looks for example patterns in a description string.
func extractExampleFromDescription(desc string) string {
	// Try "(e.g. xxx)" pattern
	for _, prefix := range []string{"e.g. ", "example: ", "e.g:"} {
		idx := strings.Index(desc, prefix)
		if idx >= 0 {
			start := idx + len(prefix)
			remaining := desc[start:]
			// Take until next punctuation or end
			end := len(remaining)
			for i, c := range remaining {
				if c == ')' || c == ',' || c == ';' || c == '\n' {
					end = i
					break
				}
			}
			return strings.TrimSpace(remaining[:end])
		}
	}

	// Try "(default: xxx)" pattern
	idx := strings.Index(desc, "default: ")
	if idx >= 0 {
		start := idx + len("default: ")
		remaining := desc[start:]
		end := len(remaining)
		for i, c := range remaining {
			if c == ')' || c == ',' || c == ';' || c == '\n' {
				end = i
				break
			}
		}
		return strings.TrimSpace(remaining[:end])
	}

	return ""
}

// isNumber checks if a value is a numeric type (float64 or int).
func isNumber(value interface{}) bool {
	switch value.(type) {
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

// isInteger checks if a value is an integer type.
func isInteger(value interface{}) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return v == float64(int(v))
	case float32:
		return float64(v) == float64(int32(v))
	default:
		return false
	}
}

// InvalidParamsResult creates an MCP CallToolResult with structured INVALID_PARAMS error.
func InvalidParamsResult(toolName string, fields []FieldError) *mcp.CallToolResult {
	details := make(map[string]interface{})
	for _, f := range fields {
		detail := map[string]string{
			"reason": f.Reason,
		}
		if f.Example != "" {
			detail["example"] = f.Example
		}
		details[f.Field] = detail
	}

	result := map[string]interface{}{
		"code":    "INVALID_PARAMS",
		"message": fmt.Sprintf("Invalid parameters for %s", toolName),
		"details": details,
	}

	content, _ := json.Marshal(result)
	return mcp.NewToolResultError(string(content))
}

// withValidation wraps a tool handler with parameter validation.
// If validation fails, a structured INVALID_PARAMS error is returned.
func withValidation(toolName string, tool mcp.Tool, handler server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		if err := ValidateParams(toolName, tool, args); err != nil {
			if ve, ok := err.(*ValidationError); ok {
				return InvalidParamsResult(toolName, ve.Fields), nil
			}
			return mcp.NewToolResultError(err.Error()), nil
		}
		return handler(ctx, request)
	}
}
