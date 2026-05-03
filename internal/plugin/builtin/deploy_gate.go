package builtin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/gin-gonic/gin"
)

// DeployGatePlugin intercepts deploy events and requires approval
// when configured to do so. It maintains an in-memory map of pending deployments.
type DeployGatePlugin struct {
	requireApproval bool
	pending         map[string]plugin.BusEvent
	mu              sync.RWMutex
}

// NewDeployGatePlugin creates a new DeployGatePlugin.
func NewDeployGatePlugin() plugin.EventPlugin {
	return &DeployGatePlugin{
		pending: make(map[string]plugin.BusEvent),
	}
}

func (p *DeployGatePlugin) Name() string {
	return "deploy-gate"
}

func (p *DeployGatePlugin) Version() string {
	return "1.0.0"
}

func (p *DeployGatePlugin) Description() string {
	return "Intercepts deploy events and holds them for approval when configured"
}

func (p *DeployGatePlugin) Init(_ context.Context, config map[string]interface{}) error {
	if config == nil {
		p.requireApproval = false
		slog.Info("deploy-gate plugin initialized (approval disabled)")
		return nil
	}
	approval, _ := config["require_approval"].(bool)
	p.requireApproval = approval
	slog.Info("deploy-gate plugin initialized",
		"require_approval", p.requireApproval,
	)
	return nil
}

func (p *DeployGatePlugin) Start() error {
	slog.Info("deploy-gate plugin started")
	return nil
}

func (p *DeployGatePlugin) Stop() error {
	slog.Info("deploy-gate plugin stopped")
	return nil
}

func (p *DeployGatePlugin) OnEvent(event plugin.BusEvent) {
	// Only handle deploy events
	if event.Type != "deploy" {
		return
	}

	if !p.requireApproval {
		return
	}

	// Store the deployment as pending
	p.mu.Lock()
	p.pending[event.ID] = event
	p.mu.Unlock()

	slog.Info("deploy-gate: deployment held for approval",
		"event_id", event.ID,
		"topic", event.Topic,
	)
}

// PendingDeployments returns all pending deployments.
func (p *DeployGatePlugin) PendingDeployments() []plugin.BusEvent {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]plugin.BusEvent, 0, len(p.pending))
	for _, evt := range p.pending {
		result = append(result, evt)
	}
	return result
}

// ApproveDeployment removes a deployment from the pending list.
func (p *DeployGatePlugin) ApproveDeployment(eventID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.pending[eventID]; !ok {
		return fmt.Errorf("deployment %s not found in pending list", eventID)
	}
	delete(p.pending, eventID)
	slog.Info("deploy-gate: deployment approved", "event_id", eventID)
	return nil
}

// RejectDeployment removes a deployment from the pending list.
func (p *DeployGatePlugin) RejectDeployment(eventID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.pending[eventID]; !ok {
		return fmt.Errorf("deployment %s not found in pending list", eventID)
	}
	delete(p.pending, eventID)
	slog.Info("deploy-gate: deployment rejected", "event_id", eventID)
	return nil
}

func (p *DeployGatePlugin) RegisterAPIRoutes(rg *gin.RouterGroup) {
	rg.GET("/pending", func(c *gin.Context) {
		pending := p.PendingDeployments()
		c.JSON(200, gin.H{
			"status": "success",
			"data":   pending,
		})
	})

	rg.POST("/approve/:event_id", func(c *gin.Context) {
		eventID := c.Param("event_id")
		if err := p.ApproveDeployment(eventID); err != nil {
			c.JSON(404, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{
			"status": "success",
			"data": gin.H{
				"message":   "deployment approved",
				"event_id":  eventID,
			},
		})
	})

	rg.POST("/reject/:event_id", func(c *gin.Context) {
		eventID := c.Param("event_id")
		if err := p.RejectDeployment(eventID); err != nil {
			c.JSON(404, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{
			"status": "success",
			"data": gin.H{
				"message":   "deployment rejected",
				"event_id":  eventID,
			},
		})
	})
}
