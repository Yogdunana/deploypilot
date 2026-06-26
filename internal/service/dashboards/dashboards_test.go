package dashboards

import (
	"testing"
)

func TestAlertHeartbeatJSON(t *testing.T) {
	result := AlertHeartbeatJSON()
	if result == nil {
		t.Fatal("AlertHeartbeatJSON() returned nil")
	}

	if uid, ok := result["uid"].(string); !ok || uid != "deploypilot-alert-heartbeat" {
		t.Errorf("Expected uid 'deploypilot-alert-heartbeat', got %v", result["uid"])
	}

	if title, ok := result["title"].(string); !ok || title != "DeployPilot - Alert & Heartbeat" {
		t.Errorf("Expected title 'DeployPilot - Alert & Heartbeat', got %v", result["title"])
	}

	if panels, ok := result["panels"].([]interface{}); !ok || len(panels) == 0 {
		t.Error("Expected panels array")
	}
}

func TestDeployStatsJSON(t *testing.T) {
	result := DeployStatsJSON()
	if result == nil {
		t.Fatal("DeployStatsJSON() returned nil")
	}

	if uid, ok := result["uid"].(string); !ok || uid != "deploypilot-deploy-stats" {
		t.Errorf("Expected uid 'deploypilot-deploy-stats', got %v", result["uid"])
	}

	if title, ok := result["title"].(string); !ok || title != "DeployPilot - Deploy Statistics" {
		t.Errorf("Expected title 'DeployPilot - Deploy Statistics', got %v", result["title"])
	}

	if panels, ok := result["panels"].([]interface{}); !ok || len(panels) == 0 {
		t.Error("Expected panels array")
	}
}

func TestServerResourcesJSON(t *testing.T) {
	result := ServerResourcesJSON()
	if result == nil {
		t.Fatal("ServerResourcesJSON() returned nil")
	}

	if uid, ok := result["uid"].(string); !ok || uid != "deploypilot-server-resources" {
		t.Errorf("Expected uid 'deploypilot-server-resources', got %v", result["uid"])
	}

	if title, ok := result["title"].(string); !ok || title != "DeployPilot - Server Resources" {
		t.Errorf("Expected title 'DeployPilot - Server Resources', got %v", result["title"])
	}

	if panels, ok := result["panels"].([]interface{}); !ok || len(panels) == 0 {
		t.Error("Expected panels array")
	}
}

func TestUptimeOverviewJSON(t *testing.T) {
	result := UptimeOverviewJSON()
	if result == nil {
		t.Fatal("UptimeOverviewJSON() returned nil")
	}

	if uid, ok := result["uid"].(string); !ok || uid != "deploypilot-uptime-overview" {
		t.Errorf("Expected uid 'deploypilot-uptime-overview', got %v", result["uid"])
	}

	if title, ok := result["title"].(string); !ok || title != "DeployPilot - Uptime Monitor" {
		t.Errorf("Expected title 'DeployPilot - Uptime Monitor', got %v", result["title"])
	}

	if panels, ok := result["panels"].([]interface{}); !ok || len(panels) == 0 {
		t.Error("Expected panels array")
	}
}