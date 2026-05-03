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
	if CheckPermission("some_unknown_tool", "viewer") {
		t.Error("unknown tools should be denied for security (H-01)")
	}
	if CheckPermission("some_unknown_tool", "dev") {
		t.Error("unknown tools should be denied for security (H-01)")
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
		level int
		want  string
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

func TestCheckPermission_ComposeOperations(t *testing.T) {
	if !CheckPermission("compose_deploy", "admin") {
		t.Error("compose_deploy requires admin level")
	}
	if !CheckPermission("compose_stop", "admin") {
		t.Error("compose_stop requires admin level")
	}
	if !CheckPermission("compose_ps", "viewer") {
		t.Error("compose_ps should be viewer accessible")
	}
	if !CheckPermission("compose_logs", "viewer") {
		t.Error("compose_logs should be viewer accessible")
	}
	if !CheckPermission("compose_restart", "dev") {
		t.Error("compose_restart requires dev level")
	}

	if CheckPermission("compose_deploy", "dev") {
		t.Error("dev should not be able to use compose_deploy")
	}
	if CheckPermission("compose_deploy", "viewer") {
		t.Error("viewer should not be able to use compose_deploy")
	}
}

func TestCheckPermission_PreflightOperations(t *testing.T) {
	if !CheckPermission("run_preflight", "viewer") {
		t.Error("run_preflight should be viewer accessible")
	}
}

func TestCheckPermission_ScheduledTasks(t *testing.T) {
	if !CheckPermission("create_scheduled_task", "admin") {
		t.Error("create_scheduled_task requires admin level")
	}
	if !CheckPermission("list_scheduled_tasks", "viewer") {
		t.Error("list_scheduled_tasks should be viewer accessible")
	}
	if !CheckPermission("get_task_executions", "viewer") {
		t.Error("get_task_executions should be viewer accessible")
	}
	if !CheckPermission("toggle_scheduled_task", "admin") {
		t.Error("toggle_scheduled_task requires admin level")
	}
	if !CheckPermission("delete_scheduled_task", "admin") {
		t.Error("delete_scheduled_task requires admin level")
	}
}

func TestCheckPermission_UptimeMonitoring(t *testing.T) {
	if !CheckPermission("create_uptime_monitor", "dev") {
		t.Error("create_uptime_monitor requires dev level")
	}
	if !CheckPermission("list_uptime_monitors", "viewer") {
		t.Error("list_uptime_monitors should be viewer accessible")
	}
	if !CheckPermission("check_uptime_monitor", "dev") {
		t.Error("check_uptime_monitor requires dev level")
	}
	if !CheckPermission("get_monitor_sla", "viewer") {
		t.Error("get_monitor_sla should be viewer accessible")
	}
	if !CheckPermission("delete_uptime_monitor", "admin") {
		t.Error("delete_uptime_monitor requires admin level")
	}
}

func TestCheckPermission_Heartbeats(t *testing.T) {
	if !CheckPermission("create_heartbeat", "dev") {
		t.Error("create_heartbeat requires dev level")
	}
	if !CheckPermission("list_heartbeats", "viewer") {
		t.Error("list_heartbeats should be viewer accessible")
	}
	if !CheckPermission("delete_heartbeat", "admin") {
		t.Error("delete_heartbeat requires admin level")
	}
}

func TestCheckPermission_PortForward(t *testing.T) {
	if !CheckPermission("port_forward", "dev") {
		t.Error("port_forward requires dev level")
	}
	if CheckPermission("port_forward", "viewer") {
		t.Error("viewer should not be able to use port_forward")
	}
}

func TestCheckPermission_ExecCommand(t *testing.T) {
	if !CheckPermission("exec_command", "admin") {
		t.Error("exec_command requires admin level")
	}
	if CheckPermission("exec_command", "dev") {
		t.Error("dev should not be able to use exec_command")
	}
}

func TestCheckPermission_K8sOperations(t *testing.T) {
	if !CheckPermission("k8s_list_deployments", "viewer") {
		t.Error("k8s_list_deployments should be viewer accessible")
	}
	if !CheckPermission("k8s_get_pods", "viewer") {
		t.Error("k8s_get_pods should be viewer accessible")
	}
	if !CheckPermission("k8s_deploy", "dev") {
		t.Error("k8s_deploy requires dev level")
	}
}

func TestCheckPermission_RegistryOperations(t *testing.T) {
	if !CheckPermission("registry_login", "dev") {
		t.Error("registry_login requires dev level")
	}
	if !CheckPermission("push_image", "dev") {
		t.Error("push_image requires dev level")
	}
	if !CheckPermission("list_registry_tags", "dev") {
		t.Error("list_registry_tags requires dev level")
	}
	if !CheckPermission("ping_registry", "dev") {
		t.Error("ping_registry requires dev level")
	}
}

func TestCheckPermission_SSLOperations(t *testing.T) {
	if !CheckPermission("request_ssl_certificate", "dev") {
		t.Error("request_ssl_certificate requires dev level")
	}
	if !CheckPermission("renew_ssl_certificate", "dev") {
		t.Error("renew_ssl_certificate requires dev level")
	}
	if !CheckPermission("delete_ssl_certificate", "admin") {
		t.Error("delete_ssl_certificate requires admin level")
	}
}

func TestCheckPermission_PluginOperations(t *testing.T) {
	if !CheckPermission("list_plugins", "dev") {
		t.Error("list_plugins requires dev level")
	}
	if !CheckPermission("manage_plugin", "dev") {
		t.Error("manage_plugin requires dev level")
	}
	if !CheckPermission("get_plugin_info", "dev") {
		t.Error("get_plugin_info requires dev level")
	}
}

func TestCheckPermission_ClearContext(t *testing.T) {
	if !CheckPermission("clear_context", "dev") {
		t.Error("clear_context requires dev level")
	}
}

func TestCheckPermission_OwnerOnlyOperations(t *testing.T) {
	if !CheckPermission("update_user_role", "owner") {
		t.Error("update_user_role requires owner level")
	}
	if !CheckPermission("delete_user", "owner") {
		t.Error("delete_user requires owner level")
	}
	if CheckPermission("update_user_role", "admin") {
		t.Error("admin should not be able to update_user_role")
	}
	if CheckPermission("delete_user", "admin") {
		t.Error("admin should not be able to delete_user")
	}
}

