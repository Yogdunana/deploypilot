package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// CredentialChecker periodically checks for expiring credentials
// and sends notifications through the bridge.
type CredentialChecker struct {
	db            *gorm.DB
	bridge        *Bridge
	checkInterval time.Duration
}

// NewCredentialChecker creates a new CredentialChecker.
// The checkInterval defaults to 6 hours if zero.
func NewCredentialChecker(db *gorm.DB, bridge *Bridge, checkInterval time.Duration) *CredentialChecker {
	if checkInterval <= 0 {
		checkInterval = 6 * time.Hour
	}
	return &CredentialChecker{
		db:            db,
		bridge:        bridge,
		checkInterval: checkInterval,
	}
}

// Start launches the credential checker in a goroutine.
// It runs periodically until the context is cancelled.
func (cc *CredentialChecker) Start(ctx context.Context) {
	slog.Info("credential checker started", "interval", cc.checkInterval)

	ticker := time.NewTicker(cc.checkInterval)
	defer ticker.Stop()

	// Run an initial check immediately
	cc.check(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("credential checker stopped")
			return
		case <-ticker.C:
			cc.check(ctx)
		}
	}
}

// check performs a single credential expiry check.
func (cc *CredentialChecker) check(ctx context.Context) {
	slog.Info("running credential expiry check")

	// Find credentials expiring within 7 days
	creds, err := model.ListExpiringCredentials(cc.db, 7*24*time.Hour)
	if err != nil {
		slog.Error("failed to list expiring credentials", "error", err)
		return
	}

	if len(creds) == 0 {
		slog.Info("no expiring credentials found")
		return
	}

	slog.Info("found expiring credentials", "count", len(creds))

	for _, cred := range creds {
		days := model.DaysUntilExpiry(&cred)
		isExpired := model.IsExpired(&cred)

		var message string
		if isExpired {
			message = fmt.Sprintf("Credential '%s' has expired", cred.Name)
		} else {
			message = fmt.Sprintf("Credential '%s' expires in %d days", cred.Name, days)
		}

		// Send notification via bridge
		_, notifyErr := cc.bridge.SendNotification(ctx, "credential_expiry", cred.Name, "", "warning", message)
		if notifyErr != nil {
			slog.Error("failed to send credential expiry notification",
				"credential", cred.Name,
				"error", notifyErr,
			)
		} else {
			slog.Info("sent credential expiry notification",
				"credential", cred.Name,
				"days_until_expiry", days,
				"expired", isExpired,
			)
		}
	}
}
