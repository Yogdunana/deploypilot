package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerDNSTools registers dns tools.
func registerDNSTools(s *server.MCPServer, d DNSManager) {
	dnsCreateTool := mcp.NewTool("add_dns_record",
		mcp.WithDescription("Create a DNS record"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name (e.g. example.com)")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Record type: A, AAAA, CNAME, TXT, MX")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Record name (e.g. @ or subdomain)")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Record value (e.g. IP address)")),
	)
	s.AddTool(dnsCreateTool, withPermissionCheck("add_dns_record", withValidation("add_dns_record", dnsCreateTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDNSCreateRecord(ctx, d, request)
	})))
	dnsDeleteTool := mcp.NewTool("delete_dns_record",
		mcp.WithDescription("Delete a DNS record"),
		mcp.WithString("record_id", mcp.Required(), mcp.Description("DNS record ID to delete")),
	)
	s.AddTool(dnsDeleteTool, withPermissionCheck("delete_dns_record", withValidation("delete_dns_record", dnsDeleteTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDNSDeleteRecord(ctx, d, request)
	})))
	dnsListTool := mcp.NewTool("list_dns_records",
		mcp.WithDescription("List DNS records for a domain"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
	)
	s.AddTool(dnsListTool, withPermissionCheck("list_dns_records", withValidation("list_dns_records", dnsListTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDNSListRecords(ctx, d, request)
	})))
	updateDNSTool := mcp.NewTool("update_dns_record",
		mcp.WithDescription("Update a DNS record"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
		mcp.WithString("subdomain", mcp.Required(), mcp.Description("Subdomain")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Record type: A, AAAA, CNAME, TXT")),
		mcp.WithString("new_value", mcp.Required(), mcp.Description("New record value")),
	)
	s.AddTool(updateDNSTool, withPermissionCheck("update_dns_record", withValidation("update_dns_record", updateDNSTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateDNSRecord(ctx, d, request)
	})))
	batchDNSTool := mcp.NewTool("batch_dns",
		mcp.WithDescription("Add multiple DNS records at once"),
		mcp.WithString("records", mcp.Required(), mcp.Description("JSON array of DNS records: [{domain, sub, type, value}]")),
	)
	s.AddTool(batchDNSTool, withPermissionCheck("batch_dns", withValidation("batch_dns", batchDNSTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBatchDNS(ctx, d, request)
	})))

}
