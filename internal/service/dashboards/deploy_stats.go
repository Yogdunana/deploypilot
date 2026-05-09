package dashboards

import "encoding/json"

// DeployStatsJSON returns the Grafana dashboard JSON for Deploy Statistics.
func DeployStatsJSON() map[string]interface{} {
	jsonStr := `{
  "uid": "deploypilot-deploy-stats",
  "title": "DeployPilot - Deploy Statistics",
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
      "title": "Deploy Count by Status",
      "type": "timeseries",
      "gridPos": {"h": 8, "w": 24, "x": 0, "y": 0},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "sum by (status) (increase(deploypilot_deploy_total[5m]))",
          "legendFormat": "{{status}}",
          "refId": "A"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "color": {"mode": "palette-classic"},
          "custom": {
            "drawStyle": "bars",
            "lineInterpolation": "smooth",
            "fillOpacity": 80,
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
      "id": 2,
      "title": "Deploy Duration",
      "type": "timeseries",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "histogram_quantile(0.50, sum(rate(deploypilot_deploy_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "P50",
          "refId": "A"
        },
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "histogram_quantile(0.95, sum(rate(deploypilot_deploy_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "P95",
          "refId": "B"
        },
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "histogram_quantile(0.99, sum(rate(deploypilot_deploy_duration_seconds_bucket[5m])) by (le))",
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
      "id": 3,
      "title": "Deploy Success Rate",
      "type": "stat",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8},
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "${datasource}"},
          "expr": "sum(increase(deploypilot_deploy_total{status=\"success\"}[24h])) / sum(increase(deploypilot_deploy_total[24h])) * 100",
          "legendFormat": "Success Rate",
          "refId": "A"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "unit": "percent",
          "min": 0,
          "max": 100,
          "color": {"mode": "thresholds"},
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {"color": "red", "value": null},
              {"color": "yellow", "value": 80},
              {"color": "green", "value": 95}
            ]
          }
        }
      },
      "options": {
        "colorMode": "background",
        "graphMode": "area",
        "justifyMode": "auto"
      }
    }
  ]
}`
	var result map[string]interface{}
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}
