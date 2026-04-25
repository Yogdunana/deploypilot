package mcp

// ToolPermissions maps tool names to minimum required role levels.
// Roles: viewer (1), dev (2), admin (3), owner (4).
var ToolPermissions = map[string]int{
	// Viewer-level (read-only)
	"list_apps": 1, "get_app_detail": 1, "list_servers": 1,
	"list_credentials": 1, "list_dns_records": 1, "list_templates": 1,
	"get_template": 1, "get_deploy_status": 1, "get_task_status": 1,
	"list_tasks": 1, "list_deployments": 1, "detect_environment": 1,
	"health_check": 1, "doctor": 1, "get_container_metrics": 1,
	"get_system_metrics": 1, "list_alerts": 1, "list_alert_rules": 1,
	"get_ci_build_status": 1, "list_ssl_certificates": 1,
	"get_context": 1, "detect_panel": 1,
	"list_clusters": 1, "k8s_list_deployments": 1, "k8s_get_pods": 1,

	// Dev-level (operations)
	"deploy_app": 2, "create_app": 2, "update_app": 2,
	"add_server": 2, "update_server": 2, "test_server": 2,
	"add_credential": 2, "update_credential": 2,
	"add_dns_record": 2, "update_dns_record": 2,
	"rollback_app": 2, "backup_database": 2, "restore_database": 2,
	"build_and_deploy": 2, "send_notification": 2,
	"heal_container": 2, "check_deploy_readiness": 2,
	"trigger_ci_build": 2, "search_app_logs": 2, "get_app_logs": 2,
	"request_ssl_certificate": 2, "renew_ssl_certificate": 2,
	"registry_login": 2, "push_image": 2, "list_registry_tags": 2, "ping_registry": 2,
	"k8s_deploy": 2,
	"list_plugins": 2, "manage_plugin": 2, "get_plugin_info": 2,

	// Admin-level (dangerous)
	"delete_app": 3, "delete_server": 3, "delete_credential": 3,
	"delete_dns_record": 3, "batch_deploy": 3, "batch_dns": 3,
	"batch_backup": 3, "delete_ssl_certificate": 3,

	// Owner-level (system)
	"update_user_role": 4, "delete_user": 4,
}

// RoleLevels maps role names to numeric levels.
var RoleLevels = map[string]int{
	"viewer": 1, "dev": 2, "admin": 3, "owner": 4,
}

// RequiredRoleName returns the human-readable role name for a given level.
func RequiredRoleName(level int) string {
	switch level {
	case 1:
		return "viewer"
	case 2:
		return "dev"
	case 3:
		return "admin"
	case 4:
		return "owner"
	default:
		return "unknown"
	}
}

// CheckPermission returns true if the user's role meets the minimum requirement
// for the given tool. Unknown tools are allowed by default.
func CheckPermission(toolName, userRole string) bool {
	required, ok := ToolPermissions[toolName]
	if !ok {
		return true // unknown tools are allowed
	}
	userLevel, ok := RoleLevels[userRole]
	if !ok {
		return false
	}
	return userLevel >= required
}
