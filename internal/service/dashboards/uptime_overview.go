package dashboards

import "encoding/json"

// UptimeOverviewJSON returns the Grafana dashboard JSON for the Uptime Monitor overview.
func UptimeOverviewJSON() map[string]interface{} {
	jsonStr := `{
  "uid": "deploypilot-uptime-overview",
  "title": "DeployPilot - Uptime Monitor",
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
      "title": "Monitor Status",
      "type": "stat",
      "gridPos": {"h": 4, "w": 8, "x": 0, "y": 0},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "count(deploypilot_monitor_up == 1)",
          "legendFormat": "Up",
          "refId": "A"
        },
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "count(deploypilot_monitor_up == 0) OR on() vector(0)",
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
      "title": "Response Latency",
      "type": "timeseries",
      "gridPos": {"h": 8, "w": 16, "x": 8, "y": 0},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "deploypilot_monitor_latency_ms",
          "legendFormat": "{{name}}",
          "refId": "A"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "unit": "ms",
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
        "legend": {"displayMode": "list", "placement": "bottom"}
      }
    },
    {
      "id": 3,
      "title": "Uptime Percentage",
      "type": "gauge",
      "gridPos": {"h": 8, "w": 8, "x": 0, "y": 4},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "avg(deploypilot_monitor_uptime_pct)",
          "legendFormat": "Avg Uptime",
          "refId": "A"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "unit": "percent",
          "min": 0,
          "max": 100,
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {"color": "red", "value": null},
              {"color": "yellow", "value": 90},
              {"color": "green", "value": 99}
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
      "id": 4,
      "title": "Monitor Status by Type",
      "type": "piechart",
      "gridPos": {"h": 8, "w": 8, "x": 0, "y": 12},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "count by (type) (deploypilot_monitor_up)",
          "legendFormat": "{{type}}",
          "refId": "A"
        }
      ],
      "options": {
        "legend": {"displayMode": "table", "placement": "right"},
        "pieType": "donut"
      }
    }
  ]
}`
	var result map[string]interface{}
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}
