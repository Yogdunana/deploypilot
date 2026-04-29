package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// MetricStore handles persistence of metrics and alert history.
type MetricStore struct {
	db *gorm.DB
}

// NewMetricStore creates a new MetricStore.
func NewMetricStore(db *gorm.DB) *MetricStore {
	return &MetricStore{db: db}
}

// SaveMetrics batch-inserts metric records.
func (s *MetricStore) SaveMetrics(ctx context.Context, metrics []Metric) error {
	if len(metrics) == 0 {
		return nil
	}
	records := make([]interface{}, 0, len(metrics))
	for _, m := range metrics {
		labels, _ := json.Marshal(m.Labels)
		records = append(records, map[string]interface{}{
			"id":          fmt.Sprintf("%d-%s-%s", m.Timestamp.UnixNano(), m.Type, m.Name),
			"metric_type": string(m.Type),
			"name":        m.Name,
			"value":       m.Value,
			"unit":        m.Unit,
			"labels":      string(labels),
			"timestamp":   m.Timestamp,
		})
	}
	return s.db.WithContext(ctx).Table("metric_records").Create(records).Error
}

// QueryMetrics retrieves metrics within a time range with optional filters.
func (s *MetricStore) QueryMetrics(ctx context.Context, opts QueryOptions) ([]map[string]interface{}, error) {
	q := s.db.WithContext(ctx).Table("metric_records").
		Where("timestamp BETWEEN ? AND ?", opts.StartTime, opts.EndTime)

	if opts.MetricType != "" {
		q = q.Where("metric_type = ?", opts.MetricType)
	}
	if opts.ServerID != "" {
		q = q.Where("server_id = ?", opts.ServerID)
	}
	if opts.ContainerName != "" {
		q = q.Where("container_name = ?", opts.ContainerName)
	}

	var results []map[string]interface{}
	err := q.Order("timestamp ASC").Limit(opts.Limit).Find(&results).Error
	return results, err
}

// SaveAlert persists an alert to history.
func (s *MetricStore) SaveAlert(ctx context.Context, alert map[string]interface{}) error {
	return s.db.WithContext(ctx).Table("alert_histories").Create(alert).Error
}

// QueryAlerts retrieves alert history with optional filters.
func (s *MetricStore) QueryAlerts(ctx context.Context, opts AlertQueryOptions) ([]map[string]interface{}, error) {
	q := s.db.WithContext(ctx).Table("alert_histories")
	if opts.Status != "" {
		q = q.Where("status = ?", opts.Status)
	}
	if opts.Severity != "" {
		q = q.Where("severity = ?", opts.Severity)
	}
	var results []map[string]interface{}
	err := q.Order("fired_at DESC").Limit(opts.Limit).Find(&results).Error
	return results, err
}

// CleanupOldMetrics deletes metrics older than the given duration.
func (s *MetricStore) CleanupOldMetrics(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	return s.db.WithContext(ctx).Exec("DELETE FROM metric_records WHERE timestamp < ?", cutoff).Error
}

// QueryOptions defines filters for metric queries.
type QueryOptions struct {
	StartTime     time.Time
	EndTime       time.Time
	MetricType    string
	ServerID      string
	ContainerName string
	Limit         int
}

// AlertQueryOptions defines filters for alert queries.
type AlertQueryOptions struct {
	Status   string
	Severity string
	Limit    int
}
