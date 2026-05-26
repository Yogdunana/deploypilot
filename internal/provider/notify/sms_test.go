package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewSMSNotifier(t *testing.T) {
	sms := NewSMSNotifier("key-id", "key-secret", "sign-name", "template-code")
	if sms.Provider != "alicloud" {
		t.Errorf("Provider = %q", sms.Provider)
	}
	if sms.AccessKeyID != "key-id" {
		t.Errorf("AccessKeyID = %q", sms.AccessKeyID)
	}
	if sms.AccessKeySecret != "key-secret" {
		t.Errorf("AccessKeySecret = %q", sms.AccessKeySecret)
	}
	if sms.SignName != "sign-name" {
		t.Errorf("SignName = %q", sms.SignName)
	}
	if sms.TemplateCode != "template-code" {
		t.Errorf("TemplateCode = %q", sms.TemplateCode)
	}
}

func TestNewGenericSMSNotifier(t *testing.T) {
	sms := NewGenericSMSNotifier("https://api.example.com/sms", "api-key", "phone")
	if sms.Provider != "generic" {
		t.Errorf("Provider = %q", sms.Provider)
	}
	if sms.GatewayURL != "https://api.example.com/sms" {
		t.Errorf("GatewayURL = %q", sms.GatewayURL)
	}
	if sms.GatewayKey != "api-key" {
		t.Errorf("GatewayKey = %q", sms.GatewayKey)
	}
	if sms.PhoneField != "phone" {
		t.Errorf("PhoneField = %q", sms.PhoneField)
	}
}

func TestSMSNotifierName(t *testing.T) {
	sms := NewSMSNotifier("id", "secret", "sign", "template")
	if sms.Name() != "sms" {
		t.Errorf("Name() = %q, want %q", sms.Name(), "sms")
	}
}

func TestSMSSendGenericSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sms := NewGenericSMSNotifier(server.URL, "test-key", "phone")
	notification := Notification{
		Type:      "deploy",
		AppName:   "my-app",
		Server:    "prod",
		Status:    "success",
		Message:   "deployed",
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"phone": "13800138000",
		},
	}

	result, err := sms.Send(context.Background(), notification)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
}

func TestSMSSendGenericWithAuth(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sms := NewGenericSMSNotifier(server.URL, "test-token", "phone")
	notification := Notification{
		Type:    "deploy",
		AppName: "my-app",
		Status:  "success",
		Message: "deployed",
	}

	_, _ = sms.Send(context.Background(), notification)

	if authHeader != "Bearer test-token" {
		t.Errorf("Authorization header = %q", authHeader)
	}
}

func TestSMSSendGenericHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sms := NewGenericSMSNotifier(server.URL, "", "")
	result, _ := sms.Send(context.Background(), DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail for 500 response")
	}
	if !strings.Contains(result.Error, "500") {
		t.Errorf("error = %q", result.Error)
	}
}

func TestSMSSendGenericConnectionRefused(t *testing.T) {
	sms := NewGenericSMSNotifier("http://127.0.0.1:1", "", "")
	result, _ := sms.Send(context.Background(), DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail for connection refused")
	}
}

func TestSMSSendUnknownProvider(t *testing.T) {
	sms := &SMSNotifier{
		Provider: "unknown",
		Client:   &http.Client{},
	}

	result, _ := sms.Send(context.Background(), DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail for unknown provider")
	}
	if !strings.Contains(result.Error, "unknown sms provider") {
		t.Errorf("error = %q", result.Error)
	}
}

func TestSMSExtractPhone(t *testing.T) {
	sms := NewSMSNotifier("id", "secret", "sign", "template")

	tests := []struct {
		name     string
		metadata map[string]string
		want     string
	}{
		{"phone key", map[string]string{"phone": "13800138000"}, "13800138000"},
		{"mobile key", map[string]string{"mobile": "13900139000"}, "13900139000"},
		{"phone first", map[string]string{"phone": "13800138000", "mobile": "13900139000"}, "13800138000"},
		{"no phone", map[string]string{"other": "value"}, ""},
		{"nil metadata", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notification := Notification{Metadata: tt.metadata}
			got := sms.extractPhone(notification)
			if got != tt.want {
				t.Errorf("extractPhone() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSMSBuildTemplateParam(t *testing.T) {
	sms := NewSMSNotifier("id", "secret", "sign", "template")

	notification := Notification{
		AppName: "my-app",
		Status:  "success",
		Message: "this is a long message that should be truncated to 50 characters",
	}

	result := sms.buildTemplateParam(notification)
	if !strings.Contains(result, "my-app") {
		t.Errorf("template param = %q", result)
	}
	if !strings.Contains(result, "success") {
		t.Errorf("template param = %q", result)
	}
	if len(result) < 10 {
		t.Errorf("template param too short: %q", result)
	}
}

func TestSMSTruncate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		maxLen  int
		wantLen int
	}{
		{"shorter than max", "hello", 10, 5},
		{"exact length", "hello", 5, 5},
		{"longer than max", "hello world", 5, 8},
		{"empty string", "", 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if len(got) != tt.wantLen {
				t.Errorf("truncate(%q, %d) len = %d, want %d", tt.input, tt.maxLen, len(got), tt.wantLen)
			}
			if tt.maxLen >= len(tt.input) && got != tt.input {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.input)
			}
			if tt.maxLen < len(tt.input) && !strings.HasSuffix(got, "...") {
				t.Errorf("truncated string should end with '...': %q", got)
			}
		})
	}
}

func TestSMSComputeSignature(t *testing.T) {
	sms := NewSMSNotifier("test-id", "test-secret", "sign", "template")

	params := map[string]string{
		"PhoneNumbers":  "13800138000",
		"SignName":      "Test",
		"TemplateCode":  "SMS_123456789",
		"Action":        "SendSms",
		"Version":       "2017-05-25",
		"Format":        "JSON",
		"AccessKeyId":   "test-id",
		"SignatureMethod": "HMAC-SHA256",
		"SignatureVersion": "1.0",
		"SignatureNonce":   "123456",
		"Timestamp":        "2024-01-01T00:00:00Z",
	}

	signature := sms.computeSignature(params)
	if len(signature) == 0 {
		t.Error("signature should not be empty")
	}
	if len(signature) != 64 {
		t.Errorf("signature length = %d, want 64", len(signature))
	}
}

func TestSMSPercentEncode(t *testing.T) {
	tests := []struct {
		input  string
		want   string
	}{
		{"hello", "hello"},
		{"hello world", "hello+world"},
		{"test&value", "test%26value"},
		{"test=value", "test%3Dvalue"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := percentEncode(tt.input)
			if got != tt.want {
				t.Errorf("percentEncode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSMSContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sms := NewGenericSMSNotifier("http://127.0.0.1:1", "", "")
	result, _ := sms.Send(ctx, DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail with cancelled context")
	}
}