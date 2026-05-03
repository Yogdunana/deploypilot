// Package main provides a seed data script for local development.
//
// Usage:
//
//	go run scripts/seed.go
//
// This creates demo data for testing: an admin user, a demo server,
// a demo application, and basic monitoring configuration.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

func main() {
	driver := flag.String("driver", "postgres", "database driver: sqlite, postgres")
	dsn := flag.String("dsn", "host=localhost port=5432 user=deploypilot password=deploypilot_dev dbname=deploypilot sslmode=disable", "database DSN")
	flag.Parse()

	fmt.Println("=== DeployPilot Seed Data Script ===")
	fmt.Printf("Driver: %s\n", *driver)
	fmt.Printf("DSN:    %s\n", *dsn)

	db, err := database.Connect(*driver, *dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Running migrations...")
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("Running default seed...")
	if err := database.Seed(db); err != nil {
		log.Fatalf("Failed to run default seed: %v", err)
	}

	fmt.Println("Creating demo data...")
	if err := seedDemoData(db); err != nil {
		log.Fatalf("Failed to seed demo data: %v", err)
	}

	fmt.Println()
	fmt.Println("=== Seed completed successfully ===")
	fmt.Println()
	fmt.Println("Admin credentials:")
	fmt.Println("  Email:    admin@deploypilot.dev")
	fmt.Println("  Password: Admin@123456")
	fmt.Println()
	fmt.Println("Demo server:")
	fmt.Println("  Name: Local Dev Server")
	fmt.Println("  Host: localhost")
	fmt.Println()
	fmt.Println("Demo application:")
	fmt.Println("  Name:    demo-app")
	fmt.Println("  Repo:    https://github.com/example/demo-app")
	fmt.Println("  Branch:  main")
}

func seedDemoData(db *gorm.DB) error {
	// ── Admin User ──────────────────────────────────────────────
	passwordHash, err := crypto.HashPassword("Admin@123456")
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	adminUser := model.User{
		ID:           "user-admin-demo",
		TenantID:     "tenant-default",
		RoleID:       "role-owner",
		Username:     "admin",
		Email:        "admin@deploypilot.dev",
		PasswordHash: passwordHash,
	}

	result := db.Where("email = ?", adminUser.Email).First(&model.User{})
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&adminUser).Error; err != nil {
			return fmt.Errorf("create admin user: %w", err)
		}
		fmt.Println("  [+] Created admin user: admin@deploypilot.dev")
	} else if result.Error != nil {
		return fmt.Errorf("check admin user: %w", result.Error)
	} else {
		fmt.Println("  [=] Admin user already exists, skipping")
	}

	// ── Demo Credential (SSH) ──────────────────────────────────
	demoCred := model.Credential{
		ID:             "cred-demo-ssh",
		TenantID:       "tenant-default",
		Name:           "Local SSH Key",
		Type:           "ssh",
		EncryptedValue: "demo-ssh-key-placeholder",
	}

	result = db.Where("id = ?", demoCred.ID).First(&model.Credential{})
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&demoCred).Error; err != nil {
			return fmt.Errorf("create demo credential: %w", err)
		}
		fmt.Println("  [+] Created demo credential: Local SSH Key")
	} else if result.Error != nil {
		return fmt.Errorf("check demo credential: %w", result.Error)
	} else {
		fmt.Println("  [=] Demo credential already exists, skipping")
	}

	// ── Demo Provider (SSH) ────────────────────────────────────
	demoProvider := model.Provider{
		ID:       "provider-demo-ssh",
		TenantID: "tenant-default",
		Type:     "ssh",
		Name:     "Local SSH Provider",
		Config:   `{"host":"localhost","port":22,"user":"root"}`,
		Enabled:  true,
	}

	result = db.Where("id = ?", demoProvider.ID).First(&model.Provider{})
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&demoProvider).Error; err != nil {
			return fmt.Errorf("create demo provider: %w", err)
		}
		fmt.Println("  [+] Created demo provider: Local SSH Provider")
	} else if result.Error != nil {
		return fmt.Errorf("check demo provider: %w", result.Error)
	} else {
		fmt.Println("  [=] Demo provider already exists, skipping")
	}

	// ── Demo Server ────────────────────────────────────────────
	demoServer := model.Server{
		ID:           "server-demo-local",
		TenantID:     "tenant-default",
		CredentialID: demoCred.ID,
		ProviderID:   demoProvider.ID,
		Name:         "Local Dev Server",
		Host:         "localhost",
		Port:         22,
		Tags:         `["dev","local","docker"]`,
		Status:       "unknown",
	}

	result = db.Where("id = ?", demoServer.ID).First(&model.Server{})
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&demoServer).Error; err != nil {
			return fmt.Errorf("create demo server: %w", err)
		}
		fmt.Println("  [+] Created demo server: Local Dev Server (localhost:22)")
	} else if result.Error != nil {
		return fmt.Errorf("check demo server: %w", result.Error)
	} else {
		fmt.Println("  [=] Demo server already exists, skipping")
	}

	// ── Demo Application ───────────────────────────────────────
	demoApp := model.App{
		ID:             "app-demo-001",
		TenantID:       "tenant-default",
		ServerID:       demoServer.ID,
		Name:           "demo-app",
		RepoURL:        "https://github.com/example/demo-app",
		Branch:         "main",
		Domain:         "demo.localhost",
		TechStack:      "docker",
		DeployMode:     "api",
		Status:         "pending",
		Environment:    "development",
		ContainerName:  "demo-app",
		EnvVars:        `{"NODE_ENV":"development","PORT":"3000"}`,
		ResourceLimits: `{"memory":"512m","cpu":"0.5"}`,
	}

	result = db.Where("id = ?", demoApp.ID).First(&model.App{})
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&demoApp).Error; err != nil {
			return fmt.Errorf("create demo app: %w", err)
		}
		fmt.Println("  [+] Created demo app: demo-app")
	} else if result.Error != nil {
		return fmt.Errorf("check demo app: %w", result.Error)
	} else {
		fmt.Println("  [=] Demo app already exists, skipping")
	}

	// ── Demo Alert Rule (CPU) ──────────────────────────────────
	cpuAlertRule := model.AlertRuleRecord{
		ID:              "alert-rule-cpu-demo",
		TenantID:        "tenant-default",
		Name:            "Demo CPU Alert",
		MetricType:      "cpu",
		Condition:       "gt",
		Threshold:       80.0,
		Severity:        "warning",
		Enabled:         true,
		CooldownSeconds: 900,
		NotifyChannels:  `["webhook"]`,
		ServerID:        demoServer.ID,
	}

	result = db.Where("id = ?", cpuAlertRule.ID).First(&model.AlertRuleRecord{})
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&cpuAlertRule).Error; err != nil {
			return fmt.Errorf("create cpu alert rule: %w", err)
		}
		fmt.Println("  [+] Created demo alert rule: CPU > 80%")
	} else if result.Error != nil {
		return fmt.Errorf("check cpu alert rule: %w", result.Error)
	} else {
		fmt.Println("  [=] Demo CPU alert rule already exists, skipping")
	}

	// ── Demo Alert Rule (Memory) ───────────────────────────────
	memAlertRule := model.AlertRuleRecord{
		ID:              "alert-rule-mem-demo",
		TenantID:        "tenant-default",
		Name:            "Demo Memory Alert",
		MetricType:      "memory",
		Condition:       "gt",
		Threshold:       85.0,
		Severity:        "warning",
		Enabled:         true,
		CooldownSeconds: 900,
		NotifyChannels:  `["webhook"]`,
		ServerID:        demoServer.ID,
	}

	result = db.Where("id = ?", memAlertRule.ID).First(&model.AlertRuleRecord{})
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&memAlertRule).Error; err != nil {
			return fmt.Errorf("create memory alert rule: %w", err)
		}
		fmt.Println("  [+] Created demo alert rule: Memory > 85%")
	} else if result.Error != nil {
		return fmt.Errorf("check memory alert rule: %w", result.Error)
	} else {
		fmt.Println("  [=] Demo Memory alert rule already exists, skipping")
	}

	return nil
}
