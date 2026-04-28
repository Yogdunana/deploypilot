package mcp

import (
	"context"
	"fmt",
	"github.com/mark3labs/mcp-go/mcp"
)
func handleListSSLCertificates(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := deployer.ListSSLCertificates(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list SSL certificates: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleRequestSSLCertificate(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	email, err := request.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.RequestSSLCertificate(ctx, domain, email)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to request SSL certificate: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleRenewSSLCertificate(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.RenewSSLCertificate(ctx, domain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to renew SSL certificate: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleDeleteSSLCertificate(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.DeleteSSLCertificate(ctx, domain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete SSL certificate: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
