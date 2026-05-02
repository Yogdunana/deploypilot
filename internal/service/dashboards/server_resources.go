package dashboards

import "encoding/json"

// ServerResourcesJSON returns the Grafana dashboard JSON for Server Resources monitoring.
func ServerResourcesJSON() map[string]interface{} {
	jsonStr := `{
  "uid": "deploypilot-server-resources",
  "title": "DeployPilot - Server Resources",
  "tags": ["deploypilot", "auto-provisioned"],
  "timezone": "browser",
  "schemaVersion": 39,
  "version": 1,
  "refresh": "30s",
  "templating": {
    "list": [
      {
        "name": "datasource",
        "type": "datasource",
        "query": "prometheus",
        "current": {},
        "options": [],
        "hide": 0
      }
    ]
  },
  "panels": [
    {
      "id": 1,
      "title": "Active Containers",
      "type": "gauge",
      "gridPos": {"h": 6, "w": 12, "x": 0, "y": 0},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "deploypilot_active_containers",
          "legendFormat": "Active",
          "refId": "A"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "color": {"mode": "thresholds"},
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {"color": "green", "value": null},
              {"color": "yellow", "value": 50},
              {"color": "red", "value": 100}
            ]
          }
        }
      },
      "options": {
        "showThresholdLabels": true,
        "showThresholdMarkers": true
      }
    },
    {
      "id": 2,
      "title": "WebSocket Connections",
      "type": "stat",
      "gridPos": {"h": 6, "w": 12, "x": 12, "y": 0},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "deploypilot_ws_connections",
          "legendFormat": "Connections",
          "refId": "A"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "color": {"mode": "thresholds"},
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {"color": "green", "value": null},
              {"color": "yellow", "value": 100},
              {"color": "red", "value": 500}
            ]
          }
        }
      },
      "options": {
        "colorMode": "background",
        "graphMode": "area",
        "justifyMode": "auto"
      }
    },
    {
      "id": 3,
      "title": "API Request Duration (P50 / P95 / P99)",
      "type": "timeseries",
      "gridPos": {"h": 8, "w": 24, "x": 0, "y": 6},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "histogram_quantile(0.50, sum(rate(deploypilot_api_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "P50",
          "refId": "A"
        },
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "histogram_quantile(0.95, sum(rate(deploypilot_api_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "P95",
          "refId": "B"
        },
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "histogram_quantile(0.99, sum(rate(deploypilot_api_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "P99",
          "refId": "C"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "unit": "s",
          "color": {"mode": "palette-classic"},
          "custom": {
            "drawStyle": "line",
            "lineInterpolation": "smooth",
            "fillOpacity": 10,
            "showPoints": "auto"
          }
        }
      },
      "options": {
        "tooltip": {"mode": "multi", "sort": "desc"},
        "legend": {"displayMode": "table", "placement": "bottom"}
      }
    },
    {
      "id": 4,
      "title": "Credential Expiry Countdown",
      "type": "stat",
      "gridPos": {"h": 6, "w": 24, "x": 0, "y": 14},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "deploypilot_credential_expiry_days",
          "legendFormat": "{{name}}",
          "refId": "A"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "unit": "d",
          "color": {"mode": "thresholds"},
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {"color": "red", "value": null},
              {"color": "yellow", "value": 7},
              {"color": "green", "value": 30}
            ]
          }
        }
      },
      "options": {
        "colorMode": "background",
        "graphMode": "none",
        "justifyMode": "auto"
      }
    }
  ]
}`
	var result map[string]interface{}
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}
