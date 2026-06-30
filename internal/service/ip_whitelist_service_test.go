package service

import (
	"testing"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIPWhitelistTestDB(t *testing.T) (*gorm.DB, *IPWhitelistService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.AutoMigrate(&model.IPWhitelist{})
	svc := NewIPWhitelistService(db)
	return db, svc
}

func TestNewIPWhitelistService(t *testing.T) {
	db, _ := setupIPWhitelistTestDB(t)
	svc := NewIPWhitelistService(db)
	if svc == nil {
		t.Fatal("NewIPWhitelistService returned nil")
	}
	if svc.db != db {
		t.Error("service db field not set correctly")
	}
}

func TestIPWhitelistService_Create_ValidCIDR(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	entry, err := svc.Create("user-1", "Test entry", "192.168.1.0/24", "tenant-1", "user-1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if entry.ID == "" {
		t.Error("entry ID should not be empty")
	}
	if entry.UserID != "user-1" {
		t.Errorf("expected user_id user-1, got %s", entry.UserID)
	}
	if entry.CIDR != "192.168.1.0/24" {
		t.Errorf("expected CIDR 192.168.1.0/24, got %s", entry.CIDR)
	}
	if entry.TenantID != "tenant-1" {
		t.Errorf("expected tenant_id tenant-1, got %s", entry.TenantID)
	}
}

func TestIPWhitelistService_Create_SingleIP(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	_, err := svc.Create("user-1", "Single IP", "192.168.1.100/32", "tenant-1", "user-1")
	if err != nil {
		t.Fatalf("Create with single IP /32 failed: %v", err)
	}
}

func TestIPWhitelistService_Create_IPv6(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	_, err := svc.Create("user-1", "IPv6", "2001:db8::/32", "tenant-1", "user-1")
	if err != nil {
		t.Fatalf("Create with IPv6 failed: %v", err)
	}
}

func TestIPWhitelistService_Create_InvalidCIDR(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	invalidCIDRs := []string{
		"",
		"not-an-ip",
		"192.168.1",
		"192.168.1.1",
		"192.168.1.0/33",
		"256.256.256.0/24",
		"192.168.1.0/-1",
	}
	for _, cidr := range invalidCIDRs {
		_, err := svc.Create("user-1", "Test", cidr, "tenant-1", "user-1")
		if err == nil {
			t.Errorf("expected error for invalid CIDR %q, got nil", cidr)
		}
	}
}

func TestIPWhitelistService_List(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	svc.Create("user-1", "Entry 1", "192.168.1.0/24", "tenant-1", "user-1")
	svc.Create("user-1", "Entry 2", "10.0.0.0/8", "tenant-1", "user-1")
	svc.Create("user-2", "Other user", "172.16.0.0/12", "tenant-1", "user-2")

	entries, err := svc.List("user-1")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries for user-1, got %d", len(entries))
	}

	entries2, err := svc.List("user-2")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries2) != 1 {
		t.Errorf("expected 1 entry for user-2, got %d", len(entries2))
	}
}

func TestIPWhitelistService_List_Empty(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	entries, err := svc.List("nonexistent")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestIPWhitelistService_Delete_Owner(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	entry, _ := svc.Create("user-1", "Test", "192.168.1.0/24", "tenant-1", "user-1")

	err := svc.Delete(entry.ID, "user-1")
	if err != nil {
		t.Fatalf("Delete by owner failed: %v", err)
	}

	entries, _ := svc.List("user-1")
	if len(entries) != 0 {
		t.Error("entry should be deleted")
	}
}

func TestIPWhitelistService_Delete_NonOwner(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	entry, _ := svc.Create("user-1", "Test", "192.168.1.0/24", "tenant-1", "user-1")

	err := svc.Delete(entry.ID, "user-2")
	if err == nil {
		t.Error("expected error when non-owner tries to delete")
	}
}

func TestIPWhitelistService_Delete_Nonexistent(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	err := svc.Delete("nonexistent-id", "user-1")
	if err == nil {
		t.Error("expected error for nonexistent entry")
	}
}

func TestIPWhitelistService_Check_NoEntries(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	if !svc.Check("192.168.1.100", "user-1") {
		t.Error("Check should return true when user has no entries (not enforced)")
	}
}

func TestIPWhitelistService_Check_MatchingCIDR(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	svc.Create("user-1", "Office", "192.168.1.0/24", "tenant-1", "user-1")

	tests := []struct {
		ip   string
		want bool
	}{
		{"192.168.1.1", true},
		{"192.168.1.254", true},
		{"192.168.1.0", true},
		{"192.168.2.1", false},
		{"10.0.0.1", false},
	}
	for _, tt := range tests {
		got := svc.Check(tt.ip, "user-1")
		if got != tt.want {
			t.Errorf("Check(%q, user-1) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestIPWhitelistService_Check_MultipleEntries(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	svc.Create("user-1", "Office", "192.168.1.0/24", "tenant-1", "user-1")
	svc.Create("user-1", "VPN", "10.0.0.0/8", "tenant-1", "user-1")

	if !svc.Check("192.168.1.50", "user-1") {
		t.Error("should match first CIDR")
	}
	if !svc.Check("10.1.2.3", "user-1") {
		t.Error("should match second CIDR")
	}
	if svc.Check("172.16.0.1", "user-1") {
		t.Error("should not match any CIDR")
	}
}

func TestIPWhitelistService_Check_InvalidIP(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	svc.Create("user-1", "Test", "192.168.1.0/24", "tenant-1", "user-1")

	if svc.Check("not-an-ip", "user-1") {
		t.Error("Check should return false for invalid IP")
	}
	if svc.Check("", "user-1") {
		t.Error("Check should return false for empty IP")
	}
}

func TestIPWhitelistService_CheckGlobal_EmptyList(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	if !svc.CheckGlobal("192.168.1.1", nil) {
		t.Error("CheckGlobal should return true for empty list")
	}
	if !svc.CheckGlobal("192.168.1.1", []string{}) {
		t.Error("CheckGlobal should return true for empty list")
	}
}

func TestIPWhitelistService_CheckGlobal_ExactIP(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	allowed := []string{"192.168.1.100", "10.0.0.1"}
	tests := []struct {
		ip   string
		want bool
	}{
		{"192.168.1.100", true},
		{"10.0.0.1", true},
		{"192.168.1.101", false},
		{"10.0.0.2", false},
	}
	for _, tt := range tests {
		got := svc.CheckGlobal(tt.ip, allowed)
		if got != tt.want {
			t.Errorf("CheckGlobal(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestIPWhitelistService_CheckGlobal_CIDR(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	allowed := []string{"192.168.1.0/24", "10.0.0.0/8"}
	tests := []struct {
		ip   string
		want bool
	}{
		{"192.168.1.1", true},
		{"192.168.1.254", true},
		{"10.1.2.3", true},
		{"192.168.2.1", false},
		{"172.16.0.1", false},
	}
	for _, tt := range tests {
		got := svc.CheckGlobal(tt.ip, allowed)
		if got != tt.want {
			t.Errorf("CheckGlobal(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestIPWhitelistService_CheckGlobal_Mixed(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	allowed := []string{"192.168.1.100", "10.0.0.0/8"}
	if !svc.CheckGlobal("192.168.1.100", allowed) {
		t.Error("should match exact IP")
	}
	if !svc.CheckGlobal("10.1.2.3", allowed) {
		t.Error("should match CIDR")
	}
	if svc.CheckGlobal("192.168.1.101", allowed) {
		t.Error("should not match")
	}
}

func TestIPWhitelistService_CheckGlobal_InvalidIP(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	allowed := []string{"192.168.1.0/24"}
	if svc.CheckGlobal("not-an-ip", allowed) {
		t.Error("CheckGlobal should return false for invalid IP")
	}
}

func TestIPWhitelistService_CheckGlobal_InvalidCIDRInList(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	allowed := []string{"invalid-cidr", "192.168.1.0/24"}
	if !svc.CheckGlobal("192.168.1.1", allowed) {
		t.Error("should still match valid CIDR even with invalid ones in list")
	}
}

func TestIPWhitelistService_IsEnforced(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	if svc.IsEnforced("user-1") {
		t.Error("IsEnforced should be false when no entries")
	}

	svc.Create("user-1", "Test", "192.168.1.0/24", "tenant-1", "user-1")

	if !svc.IsEnforced("user-1") {
		t.Error("IsEnforced should be true when entries exist")
	}

	if svc.IsEnforced("user-2") {
		t.Error("IsEnforced should be false for user with no entries")
	}
}

func TestContainsSlash(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"192.168.1.0/24", true},
		{"192.168.1.1", false},
		{"", false},
		{"/", true},
		{"a/b/c", true},
	}
	for _, tt := range tests {
		got := containsSlash(tt.s)
		if got != tt.want {
			t.Errorf("containsSlash(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}
