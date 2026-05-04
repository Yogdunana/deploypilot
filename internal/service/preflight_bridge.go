package service

import (
	"context"
	"fmt"
	"log/slog"
)

func (b *Bridge) RunPreflightFull(ctx context.Context, serverID string, portMappings string) (interface{}, error) {
	var executor CommandExecutor
	var host string
	var port int

	if serverID != "" {
		remoteExec, err := b.getRemoteExecutor(ctx, serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to get remote executor for server %s: %w", serverID, err)
		}
		defer func() {
			if cerr := remoteExec.Close(); cerr != nil {
				slog.Warn("failed to close remote executor", "error", cerr)
			}
		}()
		executor = remoteExec

		// Look up server host/port for TCP check
		row := make(map[string]interface{})
		if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err == nil {
			host = toString(row["host"])
			port = toInt(row["port"])
		}
	} else {
		executor = b.Executor
	}

	cfg := PreflightConfig{
		Host:         host,
		Port:         port,
		Executor:     executor,
		PortMappings: portMappings,
	}

	result := RunPreflightFull(ctx, cfg)
	return result, nil
}

// ---------- Phase 3.3: ExecCommand ----------

