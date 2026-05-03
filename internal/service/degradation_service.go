package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/license"
	"github.com/Yogdunana/deploypilot/internal/model"
)

// DegradationLevel represents the current degradation state.
type DegradationLevel string

const (
	DegradationNone    DegradationLevel = "none"     // Fully operational
	DegradationPartial DegradationLevel = "partial"  // Some features gated
	DegradationReadOnly DegradationLevel = "readonly" // Read-only mode
)

// DegradationStatus represents the overall degradation status of the instance.
type DegradationStatus struct {
	Level           DegradationLevel `json:"level"`
	LicenseStatus   string           `json:"license_status"`
	TrialStatus     string           `json:"trial_status"`
	Tier            string           `json:"tier"`
	GatedFeatures   []string         `json:"gated_features"`
	ReadOnlyReason  string           `json:"read_only_reason,omitempty"`
	ExpiresAt       string           `json:"expires_at,omitempty"`
	GraceDaysLeft   int              `json:"grace_days_left"`
}

// GetDegradationStatus returns the current degradation status.
func (b *Bridge) GetDegradationStatus(ctx context.Context) (interface{}, error) {
	status := DegradationStatus{
		Level:         DegradationNone,
		LicenseStatus: "none",
		TrialStatus:   "none",
		Tier:          "community",
		GatedFeatures: []string{},
	}

	// 1. Check license
	if b.LicenseEngine != nil {
		info := b.LicenseEngine.GetInfo()
		if info != nil {
			status.Tier = string(info.Tier)

			if info.ValidTo.IsZero() {
				status.LicenseStatus = "active"
				status.ExpiresAt = ""
			} else {
				status.ExpiresAt = info.ValidTo.Format(time.RFC3339)
				now := time.Now()
				daysLeft := int(info.ValidTo.Sub(now).Hours() / 24)

				if now.After(info.ValidTo) {
					status.LicenseStatus = "expired"
					status.GraceDaysLeft = 0
					status.Level = DegradationReadOnly
					status.ReadOnlyReason = fmt.Sprintf("License expired on %s. Please renew your license.", info.ValidTo.Format("2006-01-02"))
				} else if daysLeft <= 7 {
					status.LicenseStatus = "expiring_soon"
					status.GraceDaysLeft = daysLeft
					status.Level = DegradationPartial
					status.ReadOnlyReason = fmt.Sprintf("License expires in %d days. Please renew to avoid service interruption.", daysLeft)
				} else {
					status.LicenseStatus = "active"
					status.GraceDaysLeft = daysLeft
				}
			}

			// Active license: check which features are gated by tier
			if status.LicenseStatus == "active" || status.LicenseStatus == "expiring_soon" {
				status.GatedFeatures = b.getGatedFeatures(ctx)
				if len(status.GatedFeatures) > 0 && status.Level == DegradationNone {
					status.Level = DegradationPartial
				}
			}
		}
	}

	// 2. Check trial (only if no license or license expired)
	if status.LicenseStatus == "none" || status.LicenseStatus == "" || status.LicenseStatus == "expired" {
		trialActive := b.IsTrialActive(ctx)
		trialExpired := b.IsTrialExpired(ctx)

		if trialExpired {
			status.TrialStatus = "expired"
			status.Level = DegradationReadOnly
			status.ReadOnlyReason = "Trial period has expired. Please activate a license."
		} else if trialActive {
			status.TrialStatus = "active"
			status.GatedFeatures = b.getGatedFeatures(ctx)
			if len(status.GatedFeatures) > 0 {
				status.Level = DegradationPartial
			}
		}
	}

	return status, nil
}

// getGatedFeatures returns a list of feature keys that are currently disabled
// based on the license tier and feature flag evaluation.
func (b *Bridge) getGatedFeatures(ctx context.Context) []string {
	var gated []string
	for _, feature := range license.AllFeatures {
		enabled, err := b.EvaluateFeature(ctx, string(feature), "")
		if err == nil && !enabled {
			gated = append(gated, string(feature))
		}
	}
	return gated
}

// CheckReadOnly verifies if the instance is in read-only mode.
// Returns an error with a user-friendly message if read-only.
func (b *Bridge) CheckReadOnly(ctx context.Context) error {
	status, err := b.GetDegradationStatus(ctx)
	if err != nil {
		return nil // fail-open
	}

	ds, ok := status.(DegradationStatus)
	if !ok {
		return nil
	}

	if ds.Level == DegradationReadOnly {
		return fmt.Errorf("instance is in read-only mode: %s", ds.ReadOnlyReason)
	}

	return nil
}

// AuditDegradation logs a degradation event to the audit trail.
func (b *Bridge) AuditDegradation(ctx context.Context, action model.DegradationAction, feature, reason, tenantID, userID, ipAddress string) {
	audit := model.DegradationAudit{
		ID:        fmt.Sprintf("da-%d", time.Now().UnixNano()),
		Action:    action,
		Feature:   feature,
		Reason:    reason,
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: ipAddress,
	}
	if err := b.DB.Create(&audit).Error; err != nil {
		slog.Warn("failed to log degradation audit", "error", err, "feature", feature, "action", action)
	}
}

// ListDegradationAudits returns recent degradation audit entries (admin only).
func (b *Bridge) ListDegradationAudits(ctx context.Context, limit int) (interface{}, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var audits []model.DegradationAudit
	if err := b.DB.Order("created_at DESC").Limit(limit).Find(&audits).Error; err != nil {
		return nil, fmt.Errorf("failed to list degradation audits: %w", err)
	}

	result := make([]map[string]interface{}, len(audits))
	for i, a := range audits {
		result[i] = map[string]interface{}{
			"id":         a.ID,
			"action":     a.Action,
			"feature":    a.Feature,
			"reason":     a.Reason,
			"tenant_id":  a.TenantID,
			"user_id":    a.UserID,
			"ip_address": a.IPAddress,
			"created_at": a.CreatedAt,
		}
	}
	return map[string]interface{}{
		"audits": result,
		"total":  len(result),
	}, nil
}

// ExportDegradationSummary returns a summary of all data for export before downgrade.
func (b *Bridge) ExportDegradationSummary(ctx context.Context) (interface{}, error) {
	summary := make(map[string]interface{})

	// Count key entities
	type countResult struct {
		Count int64
	}

	var appCount countResult
	b.DB.Table("apps").Count(&appCount.Count)
	summary["apps"] = appCount.Count

	var serverCount countResult
	b.DB.Table("servers").Count(&serverCount.Count)
	summary["servers"] = serverCount.Count

	var deployCount countResult
	b.DB.Table("deployments").Count(&deployCount.Count)
	summary["deployments"] = deployCount.Count

	var licenseCount countResult
	b.DB.Table("licenses").Count(&licenseCount.Count)
	summary["licenses"] = licenseCount.Count

	var userCount countResult
	b.DB.Table("users").Count(&userCount.Count)
	summary["users"] = userCount.Count

	summary["exported_at"] = time.Now().Format(time.RFC3339)
	summary["machine_id"] = GenerateMachineID()

	return summary, nil
}
