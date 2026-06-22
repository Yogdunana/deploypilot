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
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.IPWhitelist{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db, NewIPWhitelistService(db)
}

func TestIPWhitelistService_Create_ValidCIDR(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	entry, err := svc.Create("user-1", "office", "192.168.1.0/24", "tenant-1", "admin")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if entry.ID == "" {
		t.Error("expected ID to be set")
	}
	if entry.CIDR != "192.168.1.0/24" {
		t.Errorf("expected CIDR to be preserved, got %q", entry.CIDR)
	}
	if entry.UserID != "user-1" {
		t.Errorf("expected user_id=user-1, got %q", entry.UserID)
	}
}

func TestIPWhitelistService_Create_InvalidCIDR(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	cases := []string{
		"not-an-ip",
		"999.999.999.999",
		"192.168.1.0",    // missing prefix length
		"192.168.1.0/33", // invalid prefix length
		"",               // empty
	}
	for _, cidr := range cases {
		_, err := svc.Create("user-1", "desc", cidr, "tenant-1", "admin")
		if err == nil {
			t.Errorf("expected error for invalid CIDR %q", cidr)
		}
	}
}

func TestIPWhitelistService_List_OnlyOwnerEntries(t *testing.T) {
	db, svc := setupIPWhitelistTestDB(t)

	if _, err := svc.Create("user-A", "A1", "10.0.0.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create A1 failed: %v", err)
	}
	if _, err := svc.Create("user-A", "A2", "10.0.1.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create A2 failed: %v", err)
	}
	if _, err := svc.Create("user-B", "B1", "10.0.2.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create B1 failed: %v", err)
	}

	entries, err := svc.List("user-A")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries for user-A, got %d", len(entries))
	}
	for _, e := range entries {
		if e.UserID != "user-A" {
			t.Errorf("foreign entry leaked: user_id=%q", e.UserID)
		}
	}

	if _, err := db.DB(); err == nil {
		// Suppress unused warning for direct DB
	}
}

func TestIPWhitelistService_Delete_OwnerEnforced(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	entry, err := svc.Create("user-A", "desc", "10.0.0.0/24", "tenant-1", "admin")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Wrong user cannot delete
	if err := svc.Delete(entry.ID, "user-B"); err == nil {
		t.Error("expected error when wrong user attempts delete")
	}

	// Owner can delete
	if err := svc.Delete(entry.ID, "user-A"); err != nil {
		t.Errorf("owner delete failed: %v", err)
	}

	// Re-deleting should fail
	if err := svc.Delete(entry.ID, "user-A"); err == nil {
		t.Error("expected error when re-deleting an already deleted entry")
	}
}

func TestIPWhitelistService_Check_EmptyWhitelistAlwaysAllowed(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	// User has no entries - any IP should be allowed (backward compatible behavior).
	if !svc.Check("203.0.113.42", "user-X") {
		t.Error("expected empty whitelist to permit access")
	}
	if !svc.Check("192.168.1.5", "user-X") {
		t.Error("expected empty whitelist to permit private IP")
	}
}

func TestIPWhitelistService_Check_CIDRMatching(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	if _, err := svc.Create("user-A", "office", "10.0.0.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	cases := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.1", true},
		{"10.0.0.255", true},
		{"10.0.0.0", true},
		{"10.0.1.1", false},
		{"192.168.1.1", false},
		{"not-an-ip", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := svc.Check(tc.ip, "user-A"); got != tc.want {
			t.Errorf("Check(%q) = %v, want %v", tc.ip, got, tc.want)
		}
	}
}

func TestIPWhitelistService_Check_MultipleCIDRs(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	if _, err := svc.Create("user-A", "a", "10.0.0.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create A failed: %v", err)
	}
	if _, err := svc.Create("user-A", "b", "172.16.0.0/16", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create B failed: %v", err)
	}

	if !svc.Check("10.0.0.5", "user-A") {
		t.Error("expected 10.0.0.5 to match first CIDR")
	}
	if !svc.Check("172.16.50.50", "user-A") {
		t.Error("expected 172.16.50.50 to match second CIDR")
	}
	if svc.Check("8.8.8.8", "user-A") {
		t.Error("expected 8.8.8.8 to be rejected")
	}
}

func TestIPWhitelistService_IsEnforced(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	// No entries yet
	if svc.IsEnforced("user-A") {
		t.Error("expected IsEnforced=false when no entries exist")
	}

	if _, err := svc.Create("user-A", "desc", "10.0.0.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !svc.IsEnforced("user-A") {
		t.Error("expected IsEnforced=true after creating an entry")
	}
	if svc.IsEnforced("user-B") {
		t.Error("expected IsEnforced=false for user with no entries")
	}
}

func TestIPWhitelistService_CheckGlobal_EmptyList(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	// Empty allowed list -> all IPs allowed
	if !svc.CheckGlobal("203.0.113.42", nil) {
		t.Error("expected CheckGlobal to allow when allowed list is nil")
	}
	if !svc.CheckGlobal("203.0.113.42", []string{}) {
		t.Error("expected CheckGlobal to allow when allowed list is empty")
	}
}

func TestIPWhitelistService_CheckGlobal_ExactMatch(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	allowed := []string{"192.168.1.10", "10.0.0.5"}
	if !svc.CheckGlobal("192.168.1.10", allowed) {
		t.Error("expected exact match to be allowed")
	}
	if svc.CheckGlobal("192.168.1.11", allowed) {
		t.Error("expected non-allowed exact IP to be rejected")
	}
}

func TestIPWhitelistService_CheckGlobal_CIDRMatch(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	allowed := []string{"10.0.0.0/24"}

	cases := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.1", true},
		{"10.0.0.255", true},
		{"10.0.1.1", false},
		{"203.0.113.1", false},
	}
	for _, tc := range cases {
		if got := svc.CheckGlobal(tc.ip, allowed); got != tc.want {
			t.Errorf("CheckGlobal(%q) = %v, want %v", tc.ip, got, tc.want)
		}
	}
}

func TestIPWhitelistService_CheckGlobal_InvalidIP(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	allowed := []string{"10.0.0.0/24"}
	if svc.CheckGlobal("not-an-ip", allowed) {
		t.Error("expected invalid IP to be rejected when allowed list is non-empty")
	}
}

func TestIPWhitelistService_CheckGlobal_InvalidCIDRSkipped(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	allowed := []string{"not-a-cidr", "10.0.0.0/24"}
	if !svc.CheckGlobal("10.0.0.5", allowed) {
		t.Error("expected valid CIDR entry to still match even when other entries are invalid")
	}
}

func TestContainsSlash(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"10.0.0.0/24", true},
		{"/etc", true},
		{"no-slash", false},
		{"", false},
		{"/", true},
	}
	for _, tc := range cases {
		if got := containsSlash(tc.s); got != tc.want {
			t.Errorf("containsSlash(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestIPWhitelistService_IPv6(t *testing.T) {
	_, svc := setupIPWhitelistTestDB(t)

	// IPv6 CIDR
	if _, err := svc.Create("user-A", "v6", "2001:db8::/32", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create IPv6 CIDR failed: %v", err)
	}
	if !svc.Check("2001:db8::1", "user-A") {
		t.Error("expected IPv6 address in range to be allowed")
	}
	if svc.Check("2001:db9::1", "user-A") {
		t.Error("expected IPv6 address outside range to be rejected")
	}
}
