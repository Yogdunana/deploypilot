package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleListSSLCertificates(ctx context.Context, d SSLService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := d.ListSSLCertificates(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list SSL certificates: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleRequestSSLCertificate(ctx context.Context, d SSLService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	email, err := request.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := d.RequestSSLCertificate(ctx, domain, email)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to request SSL certificate: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleRenewSSLCertificate(ctx context.Context, d SSLService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := d.RenewSSLCertificate(ctx, domain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to renew SSL certificate: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleDeleteSSLCertificate(ctx context.Context, d SSLService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := d.DeleteSSLCertificate(ctx, domain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete SSL certificate: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
