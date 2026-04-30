package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerSSLTools registers ssl tools.
func registerSSLTools(s *server.MCPServer, d SSLService) {
	listSSLCertsTool := mcp.NewTool("list_ssl_certificates",
		mcp.WithDescription("List all SSL certificates"),
	)
	s.AddTool(listSSLCertsTool, withPermissionCheck("list_ssl_certificates", withValidation("list_ssl_certificates", listSSLCertsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListSSLCertificates(ctx, d, request)
	})))
	requestSSLCertTool := mcp.NewTool("request_ssl_certificate",
		mcp.WithDescription("Request a new SSL certificate for a domain"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
		mcp.WithString("email", mcp.Required(), mcp.Description("Email for certificate registration")),
	)
	s.AddTool(requestSSLCertTool, withPermissionCheck("request_ssl_certificate", withValidation("request_ssl_certificate", requestSSLCertTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRequestSSLCertificate(ctx, d, request)
	})))
	renewSSLCertTool := mcp.NewTool("renew_ssl_certificate",
		mcp.WithDescription("Renew an SSL certificate"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
	)
	s.AddTool(renewSSLCertTool, withPermissionCheck("renew_ssl_certificate", withValidation("renew_ssl_certificate", renewSSLCertTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRenewSSLCertificate(ctx, d, request)
	})))
	deleteSSLCertTool := mcp.NewTool("delete_ssl_certificate",
		mcp.WithDescription("Delete an SSL certificate"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
	)
	s.AddTool(deleteSSLCertTool, withPermissionCheck("delete_ssl_certificate", withValidation("delete_ssl_certificate", deleteSSLCertTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteSSLCertificate(ctx, d, request)
	})))

}
