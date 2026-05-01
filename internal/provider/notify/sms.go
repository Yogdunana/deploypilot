package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// SMSNotifier sends notifications via SMS using Alibaba Cloud SMS API.
// See: https://help.aliyun.com/document_detail/101414.html
// Also supports a generic HTTP SMS gateway mode.
type SMSNotifier struct {
	Provider    string // "alicloud" or "generic"
	// Alibaba Cloud SMS
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	// Generic HTTP SMS
	GatewayURL string
	GatewayKey string
	PhoneField string // JSON field name for phone number in generic gateway
	Client     *http.Client
}

// NewSMSNotifier creates a new SMSNotifier with Alibaba Cloud configuration.
func NewSMSNotifier(accessKeyID, accessKeySecret, signName, templateCode string) *SMSNotifier {
	return &SMSNotifier{
		Provider:        "alicloud",
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		SignName:        signName,
		TemplateCode:    templateCode,
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// NewGenericSMSNotifier creates a new SMSNotifier with generic HTTP gateway.
func NewGenericSMSNotifier(gatewayURL, gatewayKey, phoneField string) *SMSNotifier {
	return &SMSNotifier{
		Provider:    "generic",
		GatewayURL:  gatewayURL,
		GatewayKey:  gatewayKey,
		PhoneField:  phoneField,
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (s *SMSNotifier) Name() string { return "sms" }

// Send sends a notification via SMS.
func (s *SMSNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	switch s.Provider {
	case "alicloud":
		return s.sendAlicloud(ctx, notification)
	case "generic":
		return s.sendGeneric(ctx, notification)
	default:
		return &NotifyResult{Provider: "sms", Success: false, Error: fmt.Sprintf("unknown sms provider: %s", s.Provider)}, nil
	}
}

// sendAlicloud sends SMS via Alibaba Cloud SMS API.
func (s *SMSNotifier) sendAlicloud(ctx context.Context, notification Notification) (*NotifyResult, error) {
	params := map[string]string{
		"PhoneNumbers":  s.extractPhone(notification),
		"SignName":      s.SignName,
		"TemplateCode":  s.TemplateCode,
		"TemplateParam": s.buildTemplateParam(notification),
		"Action":        "SendSms",
		"Version":       "2017-05-25",
		"Format":        "JSON",
	}

	// Add common parameters
	params["AccessKeyId"] = s.AccessKeyID
	params["SignatureMethod"] = "HMAC-SHA256"
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = fmt.Sprintf("%d", time.Now().UnixNano())
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Compute signature
	signature := s.computeSignature(params)
	params["Signature"] = signature

	// Build request
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	apiURL := "https://dysmsapi.aliyuncs.com/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return &NotifyResult{Provider: "sms", Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.Client.Do(req)
	if err != nil {
		return &NotifyResult{Provider: "sms", Success: false, Error: err.Error()}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &NotifyResult{Provider: "sms", Success: false, Error: fmt.Sprintf("decode response failed: %v", err)}, nil
	}

	if code, _ := result["Code"].(string); code == "OK" {
		return &NotifyResult{Provider: "sms", Success: true, Message: "SMS sent via Alibaba Cloud"}, nil
	}

	errMsg, _ := result["Message"].(string)
	return &NotifyResult{
		Provider: "sms",
		Success:  false,
		Error:    fmt.Sprintf("alicloud SMS error: %s", errMsg),
	}, nil
}

// sendGeneric sends SMS via a generic HTTP gateway.
func (s *SMSNotifier) sendGeneric(ctx context.Context, notification Notification) (*NotifyResult, error) {
	payload := map[string]interface{}{
		"message": notification.Message,
		"type":    notification.Type,
		"status":  notification.Status,
	}
	if s.PhoneField != "" {
		payload[s.PhoneField] = s.extractPhone(notification)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return &NotifyResult{Provider: "sms", Success: false, Error: err.Error()}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.GatewayURL, bytes.NewReader(data))
	if err != nil {
		return &NotifyResult{Provider: "sms", Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	if s.GatewayKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.GatewayKey)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return &NotifyResult{Provider: "sms", Success: false, Error: err.Error()}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &NotifyResult{Provider: "sms", Success: true, Message: "SMS sent via generic gateway"}, nil
	}

	return &NotifyResult{
		Provider: "sms",
		Success:  false,
		Error:    fmt.Sprintf("generic SMS gateway error: HTTP %d", resp.StatusCode),
	}, nil
}

// computeSignature computes Alibaba Cloud API signature.
func (s *SMSNotifier) computeSignature(params map[string]string) string {
	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build canonical query string
	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(percentEncode(k))
		buf.WriteByte('=')
		buf.WriteString(percentEncode(params[k]))
	}
	stringToSign := "POST&%2F&" + percentEncode(buf.String())

	// HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(s.AccessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return hex.EncodeToString(mac.Sum(nil))
}

// percentEncode URL-encodes a string per Alibaba Cloud spec.
func percentEncode(s string) string {
	return url.QueryEscape(s)
}

// extractPhone extracts phone number from notification metadata.
func (s *SMSNotifier) extractPhone(n Notification) string {
	if phone, ok := n.Metadata["phone"]; ok {
		return phone
	}
	if phone, ok := n.Metadata["mobile"]; ok {
		return phone
	}
	return ""
}

// buildTemplateParam builds JSON template parameter for Alibaba Cloud SMS.
func (s *SMSNotifier) buildTemplateParam(n Notification) string {
	params := map[string]string{
		"app":    n.AppName,
		"status": n.Status,
		"msg":    truncate(n.Message, 50),
	}
	data, _ := json.Marshal(params)
	return string(data)
}

// truncate truncates a string to maxLen.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
