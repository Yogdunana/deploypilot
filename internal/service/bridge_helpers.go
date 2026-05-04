package service

import (
	"fmt"
	"log/slog"
	"time"
)

func generateID() string {
	return fmt.Sprintf("dep-%d", time.Now().UnixNano())
}

// logPreflightResult logs structured preflight check results.
func logPreflightResult(containerName string, result *PreflightResult) {
	slog.Info("preflight result", "container", containerName, "passed", result.Passed, "code", result.Code, "message", result.Message)
	for _, c := range result.Checks {
		slog.Info("preflight check", "name", c.Name, "passed", c.Passed, "message", c.Message, "suggestion", c.Suggestion)
	}
}


func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toStringOrDefault(v interface{}, def string) string {
	s := toString(v)
	if s == "" {
		return def
	}
	return s
}

func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

func defaultVal(val, def string) string {
	if val == "" {
		return def
	}
	return val
}


// ---------- GetServersByTags ----------

// ---------- Phase 3.1: Compose Operations ----------

// ComposeDeploy deploys an app using docker-compose.
