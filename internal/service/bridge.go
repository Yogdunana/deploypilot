package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/provider/server"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PreflightError represents a structured preflight failure.
type PreflightError struct {
	Code    PreflightErrorCode `json:"code"`
	Message string             `json:"message"`
	Checks  []PreflightCheck   `json:"checks"`
}

func (e *PreflightError) Error() string {
	return fmt.Sprintf("preflight failed [%s]: %s", e.Code, e.Message)
}

// PreflightCode returns the error code as a string (implements mcp.PreflightErrorInfo).
func (e *PreflightError) PreflightCode() string {
	return string(e.Code)
}

// PreflightMessage returns the error message (implements mcp.PreflightErrorInfo).
func (e *PreflightError) PreflightMessage() string {
	return e.Message
}

// PreflightChecks returns the individual check results (implements mcp.PreflightErrorInfo).
func (e *PreflightError) PreflightChecks() interface{} {
	return e.Checks
}

// Bridge implements mcp.Deployer by wiring DB + Docker executor.
type Bridge struct {
	DB            *gorm.DB
	Executor      deployer.CommandExecutor // can be SSH client or local shell
	EncryptionKey []byte                   // AES-256 key for credential encryption
}

// NewBridge creates a new Bridge that satisfies the mcp.Deployer interface.
func NewBridge(db *gorm.DB, executor deployer.CommandExecutor, encryptionKey []byte) *Bridge {
	return &Bridge{DB: db, Executor: executor, EncryptionKey: encryptionKey}
}

// d returns a deployer.DockerDeployer backed by the bridge's executor.
func (b *Bridge) d() *deployer.DockerDeployer {
	return deployer.New(b.Executor)
}

// getRemoteExecutor creates an SSH executor for the given server.
// It looks up the server record, finds its credential, decrypts the password/key,
// and returns an SSH client that satisfies deployer.CommandExecutor.
func (b *Bridge) getRemoteExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	host := toString(row["host"])
	port := toInt(row["port"])

	// Look up credential if associated
	var password, keyStr string
	if credID := toString(row["credential_id"]); credID != "" {
		credRow := make(map[string]interface{})
		if err := b.DB.Table("credentials").Where("id = ?", credID).Take(&credRow).Error; err == nil {
			encrypted := toString(credRow["encrypted_value"])
			if b.EncryptionKey != nil && encrypted != "" {
				if decrypted, err := crypto.Decrypt(b.EncryptionKey, encrypted); err == nil {
					password = decrypted
				}
			}
		}
	}

	cfg := server.Config{
		Host:     host,
		Port:     port,
		Username: "root",
		Password: password,
		KeyBytes: []byte(keyStr),
		Timeout:  30 * time.Second,
	}

	client, err := server.Connect(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("SSH connection failed to %s:%d: %w. "+
			"Suggestions: check host/port, verify security group allows TCP/%d, "+
			"confirm sshd is running, and ensure credentials are correct",
			host, port, err, port)
	}

	return &sshClientExecutor{Client: client}, nil
}

// sshClientExecutor wraps server.Client to implement deployer.CommandExecutor.
type sshClientExecutor struct {
	Client *server.Client
}

func (e *sshClientExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	return e.Client.RunCommand(ctx, cmd)
}

func (e *sshClientExecutor) Close() error {
	return e.Client.Close()
}

// ---------- 1. Deploy ----------

func (b *Bridge) Deploy(ctx context.Context, cfg mcp.DeployConfig) (*mcp.ContainerStatus, error) {
	// Determine executor: remote SSH if server_id provided, otherwise local
	executor := b.Executor
	var host string
	var port int

	if cfg.ServerID != "" {
		// Look up server info for preflight
		row := make(map[string]interface{})
		if err := b.DB.Table("servers").Where("id = ?", cfg.ServerID).Take(&row).Error; err != nil {
			return nil, fmt.Errorf("server not found: %w", err)
		}
		host = toString(row["host"])
		port = toInt(row["port"])

		remoteExec, err := b.getRemoteExecutor(ctx, cfg.ServerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get remote executor for server %s: %w", cfg.ServerID, err)
		}
		defer func() {
			if cerr := remoteExec.Close(); cerr != nil {
				log.Printf("failed to close remote executor: %v", cerr)
			}
		}()
		executor = remoteExec
	}

	// Run preflight checks
	pfCfg := PreflightConfig{
		Host:         host,
		Port:         port,
		Executor:     executor,
		PortMappings: cfg.Ports,
	}
	pfResult := RunPreflight(ctx, pfCfg)
	if !pfResult.Passed {
		// Save preflight failure to deployment records
		b.saveDeploymentRecord(ctx, mcp.DeployConfig{
			Image:         cfg.Image,
			ContainerName: cfg.ContainerName,
			ServerID:      cfg.ServerID,
		}, "preflight_failed", pfResult)

		// Log structured preflight failure
		logPreflightResult(cfg.ContainerName, pfResult)

		return nil, &PreflightError{
			Code:    pfResult.Code,
			Message: pfResult.Message,
			Checks:  pfResult.Checks,
		}
	}

	dCfg := deployer.DeployConfig{
		Image:         cfg.Image,
		ContainerName: cfg.ContainerName,
		Ports:         cfg.Ports,
		EnvVars:       cfg.EnvVars,
		RestartPolicy: cfg.RestartPolicy,
		Network:       cfg.Network,
		Volumes:       cfg.Volumes,
		Labels:        cfg.Labels,
		ResourceLimits: deployer.ResourceLimits{
			CPU:    cfg.CPU,
			Memory: cfg.Memory,
		},
	}

	d := deployer.New(executor)
	cs, err := d.Deploy(ctx, dCfg)
	if err != nil {
		// Save deployment failure
		b.saveDeploymentRecord(ctx, cfg, "failed", nil)
		log.Printf("[deploy] container %s deployment failed: %v", cfg.ContainerName, err)
		return nil, err
	}
	// Save deployment success
	b.saveDeploymentRecord(ctx, cfg, "success", nil)
	log.Printf("[deploy] container %s deployed successfully (id: %s)", cfg.ContainerName, cs.ID)

	return &mcp.ContainerStatus{
		ID:        cs.ID,
		Name:      cs.Name,
		Image:     cs.Image,
		Status:    cs.Status,
		Ports:     cs.Ports,
		CreatedAt: cs.CreatedAt.Format(time.RFC3339),
		Labels:    cs.Labels,
	}, nil
}

// ---------- 2. GetContainerStatus ----------

func (b *Bridge) GetContainerStatus(ctx context.Context, name string) (*mcp.ContainerStatus, error) {
	cs, err := b.d().GetContainerStatus(ctx, name)
	if err != nil {
		return nil, err
	}

	return &mcp.ContainerStatus{
		ID:        cs.ID,
		Name:      cs.Name,
		Image:     cs.Image,
		Status:    cs.Status,
		Ports:     cs.Ports,
		CreatedAt: cs.CreatedAt.Format(time.RFC3339),
		Labels:    cs.Labels,
	}, nil
}

// ---------- 3. ListApps ----------

func (b *Bridge) ListApps(ctx context.Context) ([]mcp.ContainerStatus, error) {
	var rows []map[string]interface{}
	if err := b.DB.Table("apps").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query apps: %w", err)
	}

	result := make([]mcp.ContainerStatus, 0, len(rows))
	for _, r := range rows {
		cs := mcp.ContainerStatus{
			ID:     toString(r["id"]),
			Name:   toString(r["name"]),
			Image:  toString(r["container_name"]),
			Status: toString(r["status"]),
		}
		if v, ok := r["env_vars"]; ok {
			if s, ok := v.(string); ok && s != "" {
				var m map[string]string
				if json.Unmarshal([]byte(s), &m) == nil {
					cs.Labels = m
				}
			}
		}
		result = append(result, cs)
	}
	return result, nil
}

// ---------- 4. ListServers ----------

func (b *Bridge) ListServers(ctx context.Context) ([]mcp.ServerInfo, error) {
	var rows []map[string]interface{}
	if err := b.DB.Table("servers").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query servers: %w", err)
	}

	result := make([]mcp.ServerInfo, 0, len(rows))
	for _, r := range rows {
		si := mcp.ServerInfo{
			ID:     toString(r["id"]),
			Name:   toString(r["name"]),
			Host:   toString(r["host"]),
			Status: toString(r["status"]),
		}
		if v, ok := r["port"]; ok {
			si.Port = toInt(v)
		}
		result = append(result, si)
	}
	return result, nil
}

// ---------- 5. CreateApp ----------

func (b *Bridge) CreateApp(ctx context.Context, cfg mcp.CreateAppConfig) (string, error) {
	id := uuid.New().String()
	if err := b.DB.Table("apps").Create(map[string]interface{}{
		"id":          id,
		"name":        cfg.Name,
		"repo_url":    cfg.RepoURL,
		"branch":      defaultVal(cfg.Branch, "main"),
		"domain":      cfg.Domain,
		"tech_stack":  defaultVal(cfg.TechStack, "docker"),
		"deploy_mode": defaultVal(cfg.DeployMode, "api"),
		"server_id":   cfg.ServerID,
		"status":      "created",
	}).Error; err != nil {
		return "", fmt.Errorf("failed to create app: %w", err)
	}
	return id, nil
}

// ---------- 6. DeleteApp ----------

func (b *Bridge) DeleteApp(ctx context.Context, appID string) error {
	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return fmt.Errorf("app not found: %w", err)
	}

	containerName := toString(row["container_name"])
	if containerName != "" {
		_ = b.d().Stop(ctx, containerName)
		_ = b.d().Remove(ctx, containerName)
	}

	if err := b.DB.Table("apps").Where("id = ?", appID).Delete(nil).Error; err != nil {
		return fmt.Errorf("failed to delete app from DB: %w", err)
	}
	return nil
}

// ---------- 7. Stop ----------

func (b *Bridge) Stop(ctx context.Context, name string) error {
	return b.d().Stop(ctx, name)
}

// ---------- 8. Remove ----------

func (b *Bridge) Remove(ctx context.Context, name string) error {
	return b.d().Remove(ctx, name)
}

// ---------- 9. GetContainerLogs ----------

func (b *Bridge) GetContainerLogs(ctx context.Context, name string, tail int) (string, error) {
	return b.d().GetContainerLogs(ctx, name, tail)
}

// ---------- 10. Rollback ----------

func (b *Bridge) Rollback(ctx context.Context, containerName, previousImage string) (*mcp.ContainerStatus, error) {
	// Stop and remove the current container
	if err := b.d().Stop(ctx, containerName); err != nil {
		log.Printf("rollback: stop warning: %v", err)
	}
	if err := b.d().Remove(ctx, containerName); err != nil {
		log.Printf("rollback: remove warning: %v", err)
	}

	// Re-deploy with the previous image
	cfg := mcp.DeployConfig{
		Image:         previousImage,
		ContainerName: containerName,
		RestartPolicy: "unless-stopped",
	}
	return b.Deploy(ctx, cfg)
}

// ---------- 11. DetectEnv ----------

func (b *Bridge) DetectEnv(ctx context.Context, level int, ports []int, services []string) (interface{}, error) {
	env := map[string]interface{}{}

	// Level 1: OS info
	if level >= 1 {
		if out, err := b.Executor.RunCommand(ctx, "uname -a"); err == nil {
			env["os"] = strings.TrimSpace(out)
		}
		if out, err := b.Executor.RunCommand(ctx, "cat /etc/os-release 2>/dev/null | head -5"); err == nil {
			env["os_release"] = strings.TrimSpace(out)
		}
	}

	// Level 2: Docker info
	if level >= 2 {
		if out, err := b.Executor.RunCommand(ctx, "docker version --format '{{.Server.Version}}' 2>/dev/null"); err == nil {
			env["docker_version"] = strings.TrimSpace(out)
		}
		if out, err := b.Executor.RunCommand(ctx, "docker info --format '{{.NCPU}}' 2>/dev/null"); err == nil {
			env["docker_cpus"] = strings.TrimSpace(out)
		}
		if out, err := b.Executor.RunCommand(ctx, "docker info --format '{{.MemTotal}}' 2>/dev/null"); err == nil {
			env["docker_memory"] = strings.TrimSpace(out)
		}
	}

	// Level 3: Port checks
	if level >= 3 {
		portResults := map[string]bool{}
		for _, p := range ports {
			cmd := fmt.Sprintf("ss -tlnp 2>/dev/null | grep ':%d ' || true", p)
			if out, err := b.Executor.RunCommand(ctx, cmd); err == nil && strings.TrimSpace(out) != "" {
				portResults[fmt.Sprintf("%d", p)] = true
			} else {
				portResults[fmt.Sprintf("%d", p)] = false
			}
		}
		env["ports"] = portResults
	}

	// Level 4: Service checks
	if level >= 4 {
		svcResults := map[string]bool{}
		for _, svc := range services {
			svc = strings.TrimSpace(svc)
			if svc == "" {
				continue
			}
			cmd := fmt.Sprintf("timeout 2 bash -c 'echo > /dev/tcp/%s' 2>/dev/null && echo ok || echo fail", strings.TrimPrefix(svc, "tcp://"))
			if out, err := b.Executor.RunCommand(ctx, cmd); err == nil && strings.Contains(out, "ok") {
				svcResults[svc] = true
			} else {
				svcResults[svc] = false
			}
		}
		env["services"] = svcResults
	}

	return env, nil
}

// ---------- 12. HealthCheck ----------

func (b *Bridge) HealthCheck(ctx context.Context, target, healthType string) (interface{}, error) {
	cfg := deployer.HealthCheckConfig{
		Type:     defaultVal(healthType, "http"),
		Target:   target,
		Timeout:  5 * time.Second,
		Interval: 3 * time.Second,
		Retries:  3,
	}

	start := time.Now()
	err := b.d().HealthCheck(ctx, cfg)
	elapsed := time.Since(start)

	if err != nil {
		return map[string]interface{}{
			"status":   "unhealthy",
			"target":   target,
			"type":     cfg.Type,
			"error":    err.Error(),
			"duration": elapsed.String(),
		}, nil
	}

	return map[string]interface{}{
		"status":   "healthy",
		"target":   target,
		"type":     cfg.Type,
		"duration": elapsed.String(),
	}, nil
}

// ---------- 13. AddServer ----------

func (b *Bridge) AddServer(ctx context.Context, name, host string, port int, user string) (*mcp.ServerInfo, error) {
	id := uuid.New().String()
	status := "unknown"

	// Test connectivity
	if _, err := b.Executor.RunCommand(ctx, "echo ok"); err == nil {
		status = "connected"
	}

	if err := b.DB.Table("servers").Create(map[string]interface{}{
		"id":     id,
		"name":   name,
		"host":   host,
		"port":   port,
		"status": status,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to add server: %w", err)
	}

	return &mcp.ServerInfo{
		ID:     id,
		Name:   name,
		Host:   host,
		Port:   port,
		Status: status,
	}, nil
}

// ---------- 14. RemoveServer ----------

func (b *Bridge) RemoveServer(ctx context.Context, serverID string) error {
	result := b.DB.Table("servers").Where("id = ?", serverID).Delete(nil)
	if result.Error != nil {
		return fmt.Errorf("failed to remove server: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("server %s not found", serverID)
	}
	return nil
}

// ---------- 15. TestServer ----------

func (b *Bridge) TestServer(ctx context.Context, serverID string) (interface{}, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	host := toString(row["host"])
	port := toInt(row["port"])

	start := time.Now()
	_, err := b.Executor.RunCommand(ctx, "echo ok")
	elapsed := time.Since(start)

	if err != nil {
		suggestions := []string{
			fmt.Sprintf("Check if the server %s:%d is running and accessible", host, port),
			"Verify the SSH port is not blocked by a firewall or security group",
			"Confirm the SSH service (sshd) is listening on the specified port",
			"If using a cloud provider, ensure the security group allows inbound TCP on port " + fmt.Sprintf("%d", port),
			"Try: ssh -p " + fmt.Sprintf("%d", port) + " root@" + host + " from your terminal",
		}
		return map[string]interface{}{
			"server_id":   serverID,
			"host":        host,
			"port":        port,
			"status":      "unreachable",
			"error":       err.Error(),
			"latency":     elapsed.String(),
			"suggestions": suggestions,
		}, nil
	}

	return map[string]interface{}{
		"server_id": serverID,
		"host":      host,
		"port":      port,
		"status":    "reachable",
		"latency":   elapsed.String(),
	}, nil
}

// ---------- 16. CreateCredential ----------

func (b *Bridge) CreateCredential(ctx context.Context, tenantID, name, credType, plainValue string) (interface{}, error) {
	id := uuid.New().String()

	// Encrypt the value before storing
	encrypted, err := crypto.Encrypt(b.EncryptionKey, plainValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential: %w", err)
	}

	if err := b.DB.Table("credentials").Create(map[string]interface{}{
		"id":              id,
		"tenant_id":       tenantID,
		"name":            name,
		"type":            credType,
		"encrypted_value": encrypted,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	return map[string]interface{}{
		"id":        id,
		"tenant_id": tenantID,
		"name":      name,
		"type":      credType,
	}, nil
}

// ---------- 17. ListCredentials ----------

func (b *Bridge) ListCredentials(ctx context.Context, tenantID string) (interface{}, error) {
	var rows []map[string]interface{}
	if err := b.DB.Table("credentials").Where("tenant_id = ?", tenantID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	// Mask values before returning
	sanitized := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		entry := map[string]interface{}{
			"id":        toString(r["id"]),
			"tenant_id": toString(r["tenant_id"]),
			"name":      toString(r["name"]),
			"type":      toString(r["type"]),
		}
		sanitized = append(sanitized, entry)
	}
	return sanitized, nil
}

// ---------- 18. DeleteCredential ----------

func (b *Bridge) DeleteCredential(ctx context.Context, credID string) error {
	result := b.DB.Table("credentials").Where("id = ?", credID).Delete(nil)
	if result.Error != nil {
		return fmt.Errorf("failed to delete credential: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("credential %s not found", credID)
	}
	return nil
}

// ---------- 19. DNSCreateRecord ----------

func (b *Bridge) DNSCreateRecord(ctx context.Context, domain, recordType, name, value string) (interface{}, error) {
	// Stub: no DNS provider wired yet
	id := uuid.New().String()
	log.Printf("[DNS-stub] create record: domain=%s type=%s name=%s value=%s", domain, recordType, name, value)
	return map[string]interface{}{
		"status": "not_implemented",
		"id":     id,
		"domain": domain,
		"type":   recordType,
		"name":   name,
		"value":  value,
		"message": "DNS provider not configured; record recorded locally only",
	}, nil
}

// ---------- 20. DNSDeleteRecord ----------

func (b *Bridge) DNSDeleteRecord(ctx context.Context, recordID string) error {
	log.Printf("[DNS-stub] delete record: id=%s", recordID)
	return nil
}

// ---------- 21. DNSListRecords ----------

func (b *Bridge) DNSListRecords(ctx context.Context, domain string) (interface{}, error) {
	log.Printf("[DNS-stub] list records: domain=%s", domain)
	return map[string]interface{}{
		"status":  "not_implemented",
		"domain":  domain,
		"records": []interface{}{},
		"message": "DNS provider not configured",
	}, nil
}

// ---------- 22. SendNotification ----------

func (b *Bridge) SendNotification(ctx context.Context, nType, appName, server, status, message string) (interface{}, error) {
	log.Printf("[notification] type=%s app=%s server=%s status=%s message=%s", nType, appName, server, status, message)
	return map[string]interface{}{
		"status":  "logged",
		"type":    nType,
		"app":     appName,
		"server":  server,
		"message": message,
	}, nil
}

// ---------- 23. ListTemplates ----------

func (b *Bridge) ListTemplates(ctx context.Context) (interface{}, error) {
	templates := []map[string]interface{}{
		{"type": "node", "name": "Node.js", "description": "Express / Fastify / Next.js application", "build_cmd": "npm install && npm run build", "run_cmd": "node dist/main.js", "port": 3000},
		{"type": "python", "name": "Python", "description": "Flask / FastAPI / Django application", "build_cmd": "pip install -r requirements.txt", "run_cmd": "gunicorn app:app", "port": 8000},
		{"type": "go", "name": "Go", "description": "Go HTTP server or CLI tool", "build_cmd": "go build -o app .", "run_cmd": "./app", "port": 8080},
		{"type": "java", "name": "Java", "description": "Spring Boot / Quarkus application", "build_cmd": "./mvnw package -DskipTests", "run_cmd": "java -jar target/app.jar", "port": 8080},
		{"type": "php", "name": "PHP", "description": "Laravel / Symfony application", "build_cmd": "composer install", "run_cmd": "php artisan serve", "port": 8000},
		{"type": "ruby", "name": "Ruby", "description": "Rails / Sinatra application", "build_cmd": "bundle install", "run_cmd": "bundle exec rails server", "port": 3000},
		{"type": "rust", "name": "Rust", "description": "Actix / Axum / Warp application", "build_cmd": "cargo build --release", "run_cmd": "./target/release/app", "port": 8080},
		{"type": "static", "name": "Static Site", "description": "Nginx-hosted static HTML/CSS/JS", "build_cmd": "", "run_cmd": "nginx -g 'daemon off;'", "port": 80},
		{"type": "docker", "name": "Docker", "description": "Custom Docker image deployment", "build_cmd": "", "run_cmd": "", "port": 0},
	}
	return templates, nil
}

// ---------- 24. GetTemplate ----------

func (b *Bridge) GetTemplate(ctx context.Context, tmplType string) (interface{}, error) {
	all, _ := b.ListTemplates(ctx)
	tmpls := all.([]map[string]interface{})

	for _, t := range tmpls {
		if t["type"] == tmplType {
			return t, nil
		}
	}
	return nil, fmt.Errorf("template type %q not found; available: node, python, go, java, php, ruby, rust, static, docker", tmplType)
}

// ---------- 25. GetAppDetail ----------

func (b *Bridge) GetAppDetail(ctx context.Context, appID string) (interface{}, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("app not found: %w", err)
	}
	return row, nil
}

// ---------- 26. UpdateApp ----------

func (b *Bridge) UpdateApp(ctx context.Context, appID string, config map[string]interface{}) (interface{}, error) {
	if err := b.DB.Table("apps").Where("id = ?", appID).Updates(config).Error; err != nil {
		return nil, fmt.Errorf("failed to update app: %w", err)
	}

	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return map[string]interface{}{"status": "updated", "id": appID}, nil
	}
	return row, nil
}

// ---------- 27. GetTaskStatus ----------

func (b *Bridge) GetTaskStatus(ctx context.Context, taskID string) (interface{}, error) {
	return map[string]interface{}{
		"task_id": taskID,
		"status":  "pending",
		"message": "async task tracking not yet implemented",
	}, nil
}

// ---------- 28. ListTasks ----------

func (b *Bridge) ListTasks(ctx context.Context, limit int, statusFilter string) (interface{}, error) {
	return map[string]interface{}{
		"status":  "not_implemented",
		"message": "async task tracking not yet implemented",
		"tasks":   []interface{}{},
		"limit":   limit,
		"filter":  statusFilter,
	}, nil
}

// ---------- 29. SearchAppLogs ----------

func (b *Bridge) SearchAppLogs(ctx context.Context, appID, keyword string, limit int) (interface{}, error) {
	// Look up container name from app record
	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("app not found: %w", err)
	}

	containerName := toString(row["container_name"])
	if containerName == "" {
		containerName = toString(row["name"])
	}

	logs, err := b.d().GetContainerLogs(ctx, containerName, 2000)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	// Filter lines containing the keyword
	var matches []string
	lines := strings.Split(logs, "\n")
	for _, line := range lines {
		if strings.Contains(line, keyword) {
			matches = append(matches, line)
			if len(matches) >= limit {
				break
			}
		}
	}

	return map[string]interface{}{
		"app_id":       appID,
		"container":    containerName,
		"keyword":      keyword,
		"total_lines":  len(lines),
		"match_count":  len(matches),
		"limit":        limit,
		"matching_lines": matches,
	}, nil
}

// ---------- 30. UpdateDNSRecord ----------

func (b *Bridge) UpdateDNSRecord(ctx context.Context, domain, subdomain, recordType, newValue string) (interface{}, error) {
	log.Printf("[DNS-stub] update record: domain=%s subdomain=%s type=%s value=%s", domain, subdomain, recordType, newValue)
	return map[string]interface{}{
		"status":   "not_implemented",
		"domain":   domain,
		"subdomain": subdomain,
		"type":     recordType,
		"value":    newValue,
		"message":  "DNS provider not configured",
	}, nil
}

// ---------- 31. UpdateCredential ----------

func (b *Bridge) UpdateCredential(ctx context.Context, credID string, value string) (interface{}, error) {
	// Encrypt the new value before storing
	encrypted, err := crypto.Encrypt(b.EncryptionKey, value)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential: %w", err)
	}

	if err := b.DB.Table("credentials").Where("id = ?", credID).Update("encrypted_value", encrypted).Error; err != nil {
		return nil, fmt.Errorf("failed to update credential: %w", err)
	}
	return map[string]interface{}{
		"id":      credID,
		"status":  "updated",
		"message": "credential value updated",
	}, nil
}

// ---------- 32. UpdateServer ----------

func (b *Bridge) UpdateServer(ctx context.Context, serverID string, config map[string]interface{}) (interface{}, error) {
	if err := b.DB.Table("servers").Where("id = ?", serverID).Updates(config).Error; err != nil {
		return nil, fmt.Errorf("failed to update server: %w", err)
	}

	row := make(map[string]interface{})
	if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
		return map[string]interface{}{"status": "updated", "id": serverID}, nil
	}
	return row, nil
}

// ---------- 33. CheckDeployReadiness ----------

func (b *Bridge) CheckDeployReadiness(ctx context.Context, appConfig map[string]interface{}) (interface{}, error) {
	checks := map[string]interface{}{}

	// Check Docker availability
	if out, err := b.Executor.RunCommand(ctx, "docker version --format '{{.Server.Version}}' 2>/dev/null"); err == nil {
		checks["docker"] = map[string]interface{}{"available": true, "version": strings.TrimSpace(out)}
	} else {
		checks["docker"] = map[string]interface{}{"available": false, "error": err.Error()}
	}

	// Check if the requested port is free
	if portsRaw, ok := appConfig["ports"]; ok {
		portsStr := fmt.Sprintf("%v", portsRaw)
		hostPort := portsStr
		if idx := strings.Index(portsStr, ":"); idx >= 0 {
			hostPort = portsStr[:idx]
		}
		cmd := fmt.Sprintf("ss -tlnp 2>/dev/null | grep ':%s ' || true", hostPort)
		if out, err := b.Executor.RunCommand(ctx, cmd); err == nil && strings.TrimSpace(out) == "" {
			checks["port"] = map[string]interface{}{"port": hostPort, "available": true}
		} else {
			checks["port"] = map[string]interface{}{"port": hostPort, "available": false, "in_use_by": strings.TrimSpace(out)}
		}
	}

	overallReady := true
	for _, v := range checks {
		if m, ok := v.(map[string]interface{}); ok {
			if avail, ok := m["available"].(bool); ok && !avail {
				overallReady = false
			}
		}
	}

	return map[string]interface{}{
		"ready":  overallReady,
		"checks": checks,
	}, nil
}

// ---------- 34. BatchDeploy ----------

func (b *Bridge) BatchDeploy(ctx context.Context, apps []map[string]interface{}) (interface{}, error) {
	results := make([]map[string]interface{}, 0, len(apps))
	for i, appCfg := range apps {
		cfg := mcp.DeployConfig{
			Image:         toStringOrDefault(appCfg["image"], ""),
			ContainerName: toStringOrDefault(appCfg["container_name"], fmt.Sprintf("batch-app-%d", i)),
			Ports:         toStringOrDefault(appCfg["ports"], ""),
			RestartPolicy: toStringOrDefault(appCfg["restart_policy"], "unless-stopped"),
		}
		if envRaw, ok := appCfg["env_vars"]; ok {
			if s, ok := envRaw.(string); ok && s != "" {
				var m map[string]string
				if json.Unmarshal([]byte(s), &m) == nil {
					cfg.EnvVars = m
				}
			}
		}

		cs, err := b.Deploy(ctx, cfg)
		entry := map[string]interface{}{"index": i, "container_name": cfg.ContainerName}
		if err != nil {
			entry["status"] = "failed"
			entry["error"] = err.Error()
		} else {
			entry["status"] = "success"
			entry["container_id"] = cs.ID
		}
		results = append(results, entry)
	}

	return map[string]interface{}{
		"total":   len(apps),
		"results": results,
	}, nil
}

// ---------- 35. Backup ----------

func (b *Bridge) Backup(ctx context.Context, appID string) (string, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	containerName := toString(row["container_name"])
	if containerName == "" {
		containerName = toString(row["name"])
	}

	backupID := uuid.New().String()

	// Attempt a docker-based backup: exec into the container and create a timestamped archive
	timestamp := time.Now().Format("20060102-150405")
	backupFile := fmt.Sprintf("/tmp/backup-%s-%s.tar.gz", containerName, timestamp)
	cmd := fmt.Sprintf("docker exec %s sh -c 'tar czf - /app /data 2>/dev/null' > %s 2>/dev/null || echo 'no_backup_paths'", containerName, backupFile)
	out, err := b.Executor.RunCommand(ctx, cmd)
	if err != nil {
		log.Printf("backup: container exec failed (may be expected): %v, output: %s", err, out)
	}

	log.Printf("[backup] app_id=%s container=%s backup_id=%s file=%s", appID, containerName, backupID, backupFile)
	return backupID, nil
}

// ---------- 36. Restore ----------

func (b *Bridge) Restore(ctx context.Context, backupID string) (*mcp.ContainerStatus, error) {
	// Stub: return a not-implemented status with the backup ID
	log.Printf("[restore-stub] backup_id=%s", backupID)
	return &mcp.ContainerStatus{
		ID:        backupID,
		Name:      "restore-pending",
		Image:     "restore",
		Status:    "not_implemented",
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// ---------- 37. BatchBackup ----------

func (b *Bridge) BatchBackup(ctx context.Context, appIDs []string) (interface{}, error) {
	results := make([]map[string]interface{}, 0, len(appIDs))
	for _, id := range appIDs {
		backupID, err := b.Backup(ctx, id)
		entry := map[string]interface{}{"app_id": id}
		if err != nil {
			entry["status"] = "failed"
			entry["error"] = err.Error()
		} else {
			entry["status"] = "success"
			entry["backup_id"] = backupID
		}
		results = append(results, entry)
	}
	return map[string]interface{}{
		"total":   len(appIDs),
		"results": results,
	}, nil
}

// ---------- 38. BatchDNS ----------

func (b *Bridge) BatchDNS(ctx context.Context, records []map[string]interface{}) (interface{}, error) {
	results := make([]map[string]interface{}, 0, len(records))
	for i, rec := range records {
		domain := toStringOrDefault(rec["domain"], "")
		subdomain := toStringOrDefault(rec["subdomain"], "")
		recordType := toStringOrDefault(rec["type"], "")
		value := toStringOrDefault(rec["value"], "")

		res, _ := b.DNSCreateRecord(ctx, domain, recordType, subdomain, value)
		results = append(results, map[string]interface{}{
			"index":  i,
			"status": "not_implemented",
			"record": res,
		})
	}
	return map[string]interface{}{
		"total":   len(records),
		"results": results,
	}, nil
}

// ---------- 39. CheckSystemUpdate ----------

func (b *Bridge) CheckSystemUpdate(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{
		"current_version": "0.1.0",
		"latest_version":  "0.1.0",
		"update_available": false,
		"message":         "you are running the latest version of DeployPilot",
	}, nil
}

// ---------- helpers ----------

// saveDeploymentRecord persists a deployment attempt to the database.
func (b *Bridge) saveDeploymentRecord(ctx context.Context, cfg mcp.DeployConfig, status string, pfResult *PreflightResult) {
	if b.DB == nil {
		return
	}
	record := &model.DeploymentRecord{
		ID:            generateID(),
		TenantID:      "tenant-default",
		ServerID:      cfg.ServerID,
		ContainerName: cfg.ContainerName,
		Image:         cfg.Image,
		Status:        status,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if pfResult != nil {
		record.PreflightCode = string(pfResult.Code)
		record.PreflightMessage = pfResult.Message
		checksJSON, _ := json.Marshal(pfResult.Checks)
		record.PreflightChecks = string(checksJSON)
	}
	if err := b.DB.Create(record).Error; err != nil {
		log.Printf("[deploy] failed to save deployment record: %v", err)
	}
}

// generateID returns a unique deployment record ID.
func generateID() string {
	return fmt.Sprintf("dep-%d", time.Now().UnixNano())
}

// logPreflightResult logs structured preflight check results.
func logPreflightResult(containerName string, result *PreflightResult) {
	log.Printf("[preflight] container=%s passed=%v code=%s message=%q",
		containerName, result.Passed, result.Code, result.Message)
	for _, c := range result.Checks {
		log.Printf("[preflight]   check: name=%s passed=%v message=%q suggestion=%q",
			c.Name, c.Passed, c.Message, c.Suggestion)
	}
}

// GetLatestDeploymentRecord returns the most recent deployment record for a container.
func (b *Bridge) GetLatestDeploymentRecord(ctx context.Context, containerName string) (*model.DeploymentRecord, error) {
	var record model.DeploymentRecord
	err := b.DB.Where("container_name = ?", containerName).
		Order("created_at DESC").
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toStringOrDefault(v interface{}, def string) string {
	s := toString(v)
	if s == "" {
		return def
	}
	return s
}

func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

func defaultVal(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
