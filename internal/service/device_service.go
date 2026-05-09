package service

import (
	"fmt"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceService handles device binding, trust management, and validation.
type DeviceService struct {
	db *gorm.DB
}

// NewDeviceService creates a new DeviceService.
func NewDeviceService(db *gorm.DB) *DeviceService {
	return &DeviceService{db: db}
}

// RegisterDevice creates a new device record or updates an existing one.
// If trustDays > 0, the device is marked as trusted for that many days.
func (s *DeviceService) RegisterDevice(userID, userAgent, ip, name string, trustDays int) (*model.Device, error) {
	deviceID := model.GenerateDeviceID(userAgent, ip)

	var existing model.Device
	result := s.db.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&existing)

	now := time.Now()

	if result.Error == nil {
		// Device exists — update LastSeenAt, LastIP, UserAgent
		existing.LastIP = ip
		existing.LastSeenAt = &now
		if userAgent != "" {
			existing.UserAgent = userAgent
		}
		if name != "" {
			existing.DeviceName = name
		}
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, fmt.Errorf("failed to update device: %w", err)
		}
		return &existing, nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query device: %w", result.Error)
	}

	// New device
	device := model.Device{
		ID:         uuid.New().String(),
		TenantID:   "tenant-default",
		UserID:     userID,
		DeviceID:   deviceID,
		DeviceName: name,
		UserAgent:  userAgent,
		IP:         ip,
		LastIP:     ip,
		LastSeenAt: &now,
	}

	if trustDays > 0 {
		device.Trusted = true
		expires := now.AddDate(0, 0, trustDays)
		device.TrustExpiresAt = &expires
	}

	if err := s.db.Create(&device).Error; err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return &device, nil
}

// ListDevices returns all devices registered for the given user.
func (s *DeviceService) ListDevices(userID string) ([]model.Device, error) {
	var devices []model.Device
	if err := s.db.Where("user_id = ?", userID).Order("last_seen_at DESC").Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	return devices, nil
}

// RevokeDevice removes trust from a device owned by the given user.
func (s *DeviceService) RevokeDevice(id, userID string) error {
	result := s.db.Model(&model.Device{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"trusted":        false,
			"trust_expires_at": nil,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to revoke device: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}

// TrustDevice marks a device as trusted for the specified number of days.
func (s *DeviceService) TrustDevice(id, userID string, days int) error {
	if days <= 0 {
		days = 30
	}
	expires := time.Now().AddDate(0, 0, days)

	result := s.db.Model(&model.Device{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"trusted":         true,
			"trust_expires_at": expires,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to trust device: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}

// ValidateDevice checks if a device is currently trusted and not expired.
func (s *DeviceService) ValidateDevice(userID, deviceID string) (bool, error) {
	var device model.Device
	if err := s.db.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to validate device: %w", err)
	}

	if !device.Trusted {
		return false, nil
	}

	if device.TrustExpiresAt != nil && device.TrustExpiresAt.Before(time.Now()) {
		return false, nil
	}

	return true, nil
}

// IsNewDevice detects whether the given deviceID is new for the user.
// Returns (isNew, device, error).
func (s *DeviceService) IsNewDevice(userID, deviceID string) (bool, *model.Device, error) {
	var device model.Device
	if err := s.db.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, nil, nil
		}
		return false, nil, fmt.Errorf("failed to check device: %w", err)
	}
	return false, &device, nil
}

// CleanupExpiredTrust removes the trusted flag from devices whose trust has expired.
func (s *DeviceService) CleanupExpiredTrust() error {
	result := s.db.Model(&model.Device{}).
		Where("trusted = ? AND trust_expires_at IS NOT NULL AND trust_expires_at < ?", true, time.Now()).
		Updates(map[string]interface{}{
			"trusted":         false,
			"trust_expires_at": nil,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired trust: %w", result.Error)
	}
	return nil
}

// UpdateLastSeen updates the LastSeenAt timestamp for a device by its ID.
func (s *DeviceService) UpdateLastSeen(deviceID string) error {
	now := time.Now()
	result := s.db.Model(&model.Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"last_seen_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to update last seen: %w", result.Error)
	}
	return nil
}
