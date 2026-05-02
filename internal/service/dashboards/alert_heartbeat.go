package dashboards

import "encoding/json"

// AlertHeartbeatJSON returns the Grafana dashboard JSON for Alert & Heartbeat monitoring.
func AlertHeartbeatJSON() map[string]interface{} {
	jsonStr := `{
  "uid": "deploypilot-alert-heartbeat",
  "title": "DeployPilot - Alert & Heartbeat",
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
      "title": "Heartbeat Status",
      "type": "stat",
      "gridPos": {"h": 6, "w": 24, "x": 0, "y": 0},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "count(deploypilot_heartbeat_up == 1)",
          "legendFormat": "Up",
          "refId": "A"
        },
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "count(deploypilot_heartbeat_up == 0) OR on() vector(0)",
          "legendFormat": "Down",
          "refId": "B"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "color": {"mode": "thresholds"},
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {"color": "red", "value": null},
              {"color": "green", "value": 1}
            ]
          }
        }
      },
      "options": {
        "colorMode": "background",
        "graphMode": "none",
        "justifyMode": "auto"
      }
    },
    {
      "id": 2,
      "title": "Monitor Check Results",
      "type": "table",
      "gridPos": {"h": 10, "w": 24, "x": 0, "y": 6},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "deploypilot_monitor_up * on(name) group_left(type, target) deploypilot_monitor_latency_ms",
          "legendFormat": "",
          "format": "table",
          "instant": true,
          "refId": "A"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "custom": {
            "align": "auto",
            "filterable": true
          },
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {"color": "green", "value": null},
              {"color": "red", "value": 0.5}
            ]
          }
        },
        "overrides": [
          {
            "matcher": {"id": "byName", "options": "Value"},
            "properties": [
              {"id": "unit", "value": "ms"}
            ]
          }
        ]
      },
      "options": {
        "showHeader": true,
        "footer": {"show": false}
      },
      "transformations": [
        {
          "id": "organize",
          "options": {
            "excludeByName": {"Time": true},
            "indexByName": {"name": 0, "type": 1, "target": 2, "Value": 3},
            "renameByName": {"name": "Monitor", "type": "Type", "target": "Target", "Value": "Latency (ms)"}
          }
        }
      ]
    }
  ]
}`
	var result map[string]interface{}
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}
