package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleDNSCreateRecord(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, _ := request.RequireString("domain")
	recordType, _ := request.RequireString("type")
	name, _ := request.RequireString("name")
	value, _ := request.RequireString("value")

	if domain == "" || recordType == "" || name == "" || value == "" {
		return mcp.NewToolResultError("domain, type, name, and value are required"), nil
	}

	record, err := deployer.DNSCreateRecord(ctx, domain, recordType, name, value)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DNS create failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "record": record}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleDNSDeleteRecord(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	recordID, err := request.RequireString("record_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := deployer.DNSDeleteRecord(ctx, recordID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DNS delete failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "message": fmt.Sprintf("Record %s deleted", recordID)}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleDNSListRecords(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	records, err := deployer.DNSListRecords(ctx, domain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DNS list failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "records": records}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleUpdateDNSRecord(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, _ := request.RequireString("domain")
	subdomain, _ := request.RequireString("subdomain")
	recordType, _ := request.RequireString("type")
	newValue, _ := request.RequireString("new_value")
	if domain == "" || subdomain == "" || recordType == "" || newValue == "" {
		return mcp.NewToolResultError("domain, subdomain, type, and new_value are required"), nil
	}
	res, err := deployer.UpdateDNSRecord(ctx, domain, subdomain, recordType, newValue)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("update DNS failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "record": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleBatchDNS(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	recordsStr, err := request.RequireString("records")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(recordsStr), &records); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid records JSON: %v", err)), nil
	}
	res, err := deployer.BatchDNS(ctx, records)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("batch DNS failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "batch": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
