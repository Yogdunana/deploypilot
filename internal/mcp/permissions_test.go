package mcp

import (
	"testing"
)

func TestCheckPermission_ViewerCanRead(t *testing.T) {
	viewerTools := []string{
		"list_apps", "get_app_detail", "list_servers",
		"list_credentials", "list_dns_records", "list_templates",
		"get_template", "get_deploy_status", "get_task_status",
		"list_tasks", "detect_environment", "health_check", "doctor",
		"get_container_metrics", "get_system_metrics",
		"list_alerts", "list_alert_rules", "get_ci_build_status",
	}
	for _, tool := range viewerTools {
		t.Run(tool, func(t *testing.T) {
			if !CheckPermission(tool, "viewer") {
				t.Errorf("viewer should be able to use %s", tool)
			}
		})
	}
}

func TestCheckPermission_DevCanDeploy(t *testing.T) {
	devTools := []string{
		"deploy_app", "create_app", "update_app",
		"add_server", "update_server", "test_server",
		"add_credential", "update_credential",
		"add_dns_record", "update_dns_record",
		"rollback_app", "backup_database", "restore_database",
		"build_and_deploy", "send_notification",
		"heal_container", "check_deploy_readiness",
		"trigger_ci_build", "search_app_logs", "get_app_logs",
	}
	for _, tool := range devTools {
		t.Run(tool, func(t *testing.T) {
			if !CheckPermission(tool, "dev") {
				t.Errorf("dev should be able to use %s", tool)
			}
		})
	}
}

func TestCheckPermission_AdminCanDelete(t *testing.T) {
	adminTools := []string{
		"delete_app", "delete_server", "delete_credential",
		"delete_dns_record", "batch_deploy", "batch_dns", "batch_backup",
	}
	for _, tool := range adminTools {
		t.Run(tool, func(t *testing.T) {
			if !CheckPermission(tool, "admin") {
				t.Errorf("admin should be able to use %s", tool)
			}
		})
	}
}

func TestCheckPermission_ViewerCannotDeploy(t *testing.T) {
	restrictedTools := []string{
		"deploy_app", "create_app", "delete_app",
		"add_server", "delete_server",
		"add_credential", "delete_credential",
		"add_dns_record", "delete_dns_record",
		"batch_deploy", "batch_dns", "batch_backup",
		"build_and_deploy", "rollback_app",
		"update_user_role", "delete_user",
	}
	for _, tool := range restrictedTools {
		t.Run(tool, func(t *testing.T) {
			if CheckPermission(tool, "viewer") {
				t.Errorf("viewer should NOT be able to use %s", tool)
			}
		})
	}
}

func TestCheckPermission_UnknownTool_Allowed(t *testing.T) {
	if !CheckPermission("some_unknown_tool", "viewer") {
		t.Error("unknown tools should be allowed for any role")
	}
	if !CheckPermission("some_unknown_tool", "dev") {
		t.Error("unknown tools should be allowed for any role")
	}
}

func TestCheckPermission_UnknownRole_Denied(t *testing.T) {
	if CheckPermission("deploy_app", "nonexistent_role") {
		t.Error("unknown role should be denied access to protected tools")
	}
	if CheckPermission("delete_app", "hacker") {
		t.Error("unknown role should be denied access to protected tools")
	}
}

func TestRoleLevels_Completeness(t *testing.T) {
	expectedLevels := map[string]int{
		"viewer": 1, "dev": 2, "admin": 3, "owner": 4,
	}
	for role, expectedLevel := range expectedLevels {
		actual, ok := RoleLevels[role]
		if !ok {
			t.Errorf("RoleLevels missing role %q", role)
			continue
		}
		if actual != expectedLevel {
			t.Errorf("RoleLevels[%q] = %d, want %d", role, actual, expectedLevel)
		}
	}
}

func TestRequiredRoleName(t *testing.T) {
	tests := []struct {
		level   int
		want    string
	}{
		{1, "viewer"},
		{2, "dev"},
		{3, "admin"},
		{4, "owner"},
		{99, "unknown"},
		{0, "unknown"},
	}
	for _, tt := range tests {
		got := RequiredRoleName(tt.level)
		if got != tt.want {
			t.Errorf("RequiredRoleName(%d) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestCheckPermission_OwnerCanDoEverything(t *testing.T) {
	for toolName := range ToolPermissions {
		if !CheckPermission(toolName, "owner") {
			t.Errorf("owner should be able to use %s", toolName)
		}
	}
}

func TestCheckPermission_AdminCanUseViewerTools(t *testing.T) {
	viewerTools := []string{
		"list_apps", "get_app_detail", "health_check", "doctor",
	}
	for _, tool := range viewerTools {
		if !CheckPermission(tool, "admin") {
			t.Errorf("admin should be able to use viewer tool %s", tool)
		}
	}
}
