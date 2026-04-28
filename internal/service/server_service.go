package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/google/uuid"
)

// ---------- 4. ListServers ----------

func (b *Bridge) ListServers(ctx context.Context) ([]mcp.ServerInfo, error) {
	var rows []map[string]interface{}
	if err := b.DB.Table("servers").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query servers: %w", err)
	}

	result := make([]mcp.ServerInfo, 0, len(rows))
	for _, r := range rows {
		si := mcp.ServerInfo{
			ID:     toString(r["id"]),
			Name:   toString(r["name"]),
			Host:   toString(r["host"]),
			Status: toString(r["status"]),
		}
		if v, ok := r["port"]; ok {
			si.Port = toInt(v)
		}
		result = append(result, si)
	}
	return result, nil
}

// ---------- 13. AddServer ----------

func (b *Bridge) AddServer(ctx context.Context, name, host string, port int, user string) (*mcp.ServerInfo, error) {
	id := uuid.New().String()
	status := "unknown"

	// Test connectivity
	if _, err := b.Executor.RunCommand(ctx, "echo ok"); err == nil {
		status = "connected"
	}

	if err := b.DB.Table("servers").Create(map[string]interface{}{
		"id":     id,
		"name":   name,
		"host":   host,
		"port":   port,
		"status": status,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to add server: %w", err)
	}

	return &mcp.ServerInfo{
		ID:     id,
		Name:   name,
		Host:   host,
		Port:   port,
		Status: status,
	}, nil
}

// ---------- 14. RemoveServer ----------

func (b *Bridge) RemoveServer(ctx context.Context, serverID string) error {
	result := b.DB.Table("servers").Where("id = ?", serverID).Delete(nil)
	if result.Error != nil {
		return fmt.Errorf("failed to remove server: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("server %s not found", serverID)
	}
	return nil
}

// ---------- 15. TestServer ----------

func (b *Bridge) TestServer(ctx context.Context, serverID string) (interface{}, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	host := toString(row["host"])
	port := toInt(row["port"])

	start := time.Now()
	_, err := b.Executor.RunCommand(ctx, "echo ok")
	elapsed := time.Since(start)

	if err != nil {
		suggestions := []string{
			fmt.Sprintf("Check if the server %s:%d is running and accessible", host, port),
			"Verify the SSH port is not blocked by a firewall or security group",
			"Confirm the SSH service (sshd) is listening on the specified port",
			"If using a cloud provider, ensure the security group allows inbound TCP on port " + fmt.Sprintf("%d", port),
			"Try: ssh -p " + fmt.Sprintf("%d", port) + " root@" + host + " from your terminal",
		}
		return map[string]interface{}{
			"server_id":   serverID,
			"host":        host,
			"port":        port,
			"status":      "unreachable",
			"error":       err.Error(),
			"latency":     elapsed.String(),
			"suggestions": suggestions,
		}, nil
	}

	return map[string]interface{}{
		"server_id": serverID,
		"host":      host,
		"port":      port,
		"status":    "reachable",
		"latency":   elapsed.String(),
	}, nil
}

// ---------- 32. UpdateServer ----------

func (b *Bridge) UpdateServer(ctx context.Context, serverID string, config map[string]interface{}) (interface{}, error) {
	if err := b.DB.Table("servers").Where("id = ?", serverID).Updates(config).Error; err != nil {
		return nil, fmt.Errorf("failed to update server: %w", err)
	}

	row := make(map[string]interface{})
	if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
		return map[string]interface{}{"status": "updated", "id": serverID}, nil
	}
	return row, nil
}

// GetRemoteExecutorForTerminal creates an SSH executor for the given server (exported for WebSocket terminal).
func (b *Bridge) GetRemoteExecutorForTerminal(ctx context.Context, serverID string) (RemoteExecutor, error) {
	return b.getRemoteExecutor(ctx, serverID)
}

// GetServersByTags filters servers by tags. Tags can be provided as a JSON array
// or comma-separated, and returns IDs of servers that have at least one matching tag.
func (b *Bridge) GetServersByTags(ctx context.Context, tenantID string, tags []string) ([]string, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	var servers []model.Server
	if err := b.DB.Where("tenant_id = ?", tenantID).Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("failed to query servers: %w", err)
	}

	// Build a set of target tags for fast lookup
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[strings.ToLower(t)] = true
	}

	var matchedIDs []string
	for _, srv := range servers {
		if srv.Tags == "" {
			continue
		}

		// Parse tags: try JSON array first, then fall back to comma-separated
		var serverTags []string
		if err := json.Unmarshal([]byte(srv.Tags), &serverTags); err != nil {
			// Fall back to comma-separated
			for _, t := range strings.Split(srv.Tags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					serverTags = append(serverTags, t)
				}
			}
		}

		for _, st := range serverTags {
			if tagSet[strings.ToLower(strings.TrimSpace(st))] {
				matchedIDs = append(matchedIDs, srv.ID)
				break
			}
		}
	}

	return matchedIDs, nil
}
