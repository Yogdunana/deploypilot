package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/mcp"
)

// mockDeployer implements batchDeployer for testing.
type mockDeployer struct {
	failIndices map[int]bool // indices that should fail
	mu          sync.Mutex
	callOrder   []int // records the order of deploySingle calls
}

func newMockDeployer(failIndices ...int) *mockDeployer {
	m := &mockDeployer{
		failIndices: make(map[int]bool),
	}
	for _, i := range failIndices {
		m.failIndices[i] = true
	}
	return m
}

func (m *mockDeployer) deploySingle(ctx context.Context, index int, appCfg map[string]interface{}) mcp.BatchDeployItemResult {
	m.mu.Lock()
	m.callOrder = append(m.callOrder, index)
	m.mu.Unlock()

	appName := fmt.Sprintf("app-%d", index)
	if m.failIndices[index] {
		return mcp.BatchDeployItemResult{
			Index:   index,
			AppName: appName,
			Success: false,
			Error:   fmt.Sprintf("deploy failed for app-%d", index),
		}
	}
	return mcp.BatchDeployItemResult{
		Index:   index,
		AppName: appName,
		Success: true,
	}
}

func makeTestApps(n int) []map[string]interface{} {
	apps := make([]map[string]interface{}, n)
	for i := 0; i < n; i++ {
		apps[i] = map[string]interface{}{
			"image":          fmt.Sprintf("test/image:%d", i),
			"container_name": fmt.Sprintf("app-%d", i),
		}
	}
	return apps
}

func TestSequential_AllSuccess(t *testing.T) {
	d := newMockDeployer()
	apps := makeTestApps(3)
	config := mcp.BatchDeployConfig{
		Apps:     apps,
		Strategy: mcp.StrategySequential,
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
	if result.Success != 3 {
		t.Errorf("expected success 3, got %d", result.Success)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed 0, got %d", result.Failed)
	}
	if len(result.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Results))
	}

	// Verify sequential order
	for i, callIdx := range d.callOrder {
		if callIdx != i {
			t.Errorf("expected call order %d, got %d", i, callIdx)
		}
	}
}

func TestParallel_PartialFailure(t *testing.T) {
	// 5 apps, indices 1 and 3 fail
	d := newMockDeployer(1, 3)
	apps := makeTestApps(5)
	config := mcp.BatchDeployConfig{
		Apps:          apps,
		Strategy:      mcp.StrategyParallel,
		MaxConcurrent: 3,
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 5 {
		t.Errorf("expected total 5, got %d", result.Total)
	}
	if result.Success != 3 {
		t.Errorf("expected success 3, got %d", result.Success)
	}
	if result.Failed != 2 {
		t.Errorf("expected failed 2, got %d", result.Failed)
	}
	if len(result.Results) != 5 {
		t.Errorf("expected 5 results, got %d", len(result.Results))
	}

	// Verify failed items
	for _, r := range result.Results {
		if r.Index == 1 || r.Index == 3 {
			if r.Success {
				t.Errorf("expected app %d to fail", r.Index)
			}
		} else {
			if !r.Success {
				t.Errorf("expected app %d to succeed", r.Index)
			}
		}
	}
}

func TestParallel_ConcurrencyLimit(t *testing.T) {
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	d := &concurrencyTrackingDeployer{
		maxConcurrent:     &maxConcurrent,
		currentConcurrent: &currentConcurrent,
	}

	apps := makeTestApps(10)
	config := mcp.BatchDeployConfig{
		Apps:          apps,
		Strategy:      mcp.StrategyParallel,
		MaxConcurrent: 3,
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 10 {
		t.Errorf("expected total 10, got %d", result.Total)
	}
	if result.Success != 10 {
		t.Errorf("expected all to succeed, got %d successes", result.Success)
	}

	// Max concurrent should not exceed the limit
	observed := maxConcurrent.Load()
	if observed > 3 {
		t.Errorf("expected max concurrent <= 3, got %d", observed)
	}
}

type concurrencyTrackingDeployer struct {
	maxConcurrent     *atomic.Int32
	currentConcurrent *atomic.Int32
}

func (d *concurrencyTrackingDeployer) deploySingle(ctx context.Context, index int, appCfg map[string]interface{}) mcp.BatchDeployItemResult {
	cur := d.currentConcurrent.Add(1)
	defer d.currentConcurrent.Add(-1)

	// Track max
	for {
		old := d.maxConcurrent.Load()
		if cur <= old || d.maxConcurrent.CompareAndSwap(old, cur) {
			break
		}
	}

	return mcp.BatchDeployItemResult{
		Index:   index,
		AppName: fmt.Sprintf("app-%d", index),
		Success: true,
	}
}

func TestRolling_BatchSize(t *testing.T) {
	d := newMockDeployer()
	apps := makeTestApps(6)
	config := mcp.BatchDeployConfig{
		Apps:      apps,
		Strategy:  mcp.StrategyRolling,
		BatchSize: 2,
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 6 {
		t.Errorf("expected total 6, got %d", result.Total)
	}
	if result.Success != 6 {
		t.Errorf("expected success 6, got %d", result.Success)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed 0, got %d", result.Failed)
	}

	// Verify all 6 were called in order (rolling is sequential within batches)
	if len(d.callOrder) != 6 {
		t.Errorf("expected 6 calls, got %d", len(d.callOrder))
	}
	for i, callIdx := range d.callOrder {
		if callIdx != i {
			t.Errorf("expected call order %d, got %d", i, callIdx)
		}
	}
}

func TestRolling_UnevenBatches(t *testing.T) {
	d := newMockDeployer()
	apps := makeTestApps(7) // 7 apps with batch_size=3 => 3,3,1
	config := mcp.BatchDeployConfig{
		Apps:      apps,
		Strategy:  mcp.StrategyRolling,
		BatchSize: 3,
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 7 {
		t.Errorf("expected total 7, got %d", result.Total)
	}
	if result.Success != 7 {
		t.Errorf("expected success 7, got %d", result.Success)
	}
}

func TestEmptyApps(t *testing.T) {
	d := newMockDeployer()
	config := mcp.BatchDeployConfig{
		Apps:     []map[string]interface{}{},
		Strategy: mcp.StrategySequential,
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 0 {
		t.Errorf("expected total 0, got %d", result.Total)
	}
	if result.Success != 0 {
		t.Errorf("expected success 0, got %d", result.Success)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed 0, got %d", result.Failed)
	}
	if result.Results != nil {
		t.Errorf("expected nil results, got %v", result.Results)
	}
}

func TestInvalidStrategy_FallbackToSequential(t *testing.T) {
	d := newMockDeployer()
	apps := makeTestApps(2)
	config := mcp.BatchDeployConfig{
		Apps:     apps,
		Strategy: "invalid_strategy",
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
	if result.Success != 2 {
		t.Errorf("expected success 2, got %d", result.Success)
	}

	// Should have been called sequentially (in order)
	for i, callIdx := range d.callOrder {
		if callIdx != i {
			t.Errorf("expected sequential order %d, got %d", i, callIdx)
		}
	}
}

func TestEmptyStrategy_DefaultsToSequential(t *testing.T) {
	d := newMockDeployer()
	apps := makeTestApps(2)
	config := mcp.BatchDeployConfig{
		Apps:     apps,
		Strategy: "", // empty
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
	if result.Success != 2 {
		t.Errorf("expected success 2, got %d", result.Success)
	}
}

func TestParallel_DefaultConcurrency(t *testing.T) {
	d := newMockDeployer()
	apps := makeTestApps(3)
	config := mcp.BatchDeployConfig{
		Apps:          apps,
		Strategy:      mcp.StrategyParallel,
		MaxConcurrent: 0, // should default to 5
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
	if result.Success != 3 {
		t.Errorf("expected success 3, got %d", result.Success)
	}
}

func TestRolling_DefaultBatchSize(t *testing.T) {
	d := newMockDeployer()
	apps := makeTestApps(4)
	config := mcp.BatchDeployConfig{
		Apps:      apps,
		Strategy:  mcp.StrategyRolling,
		BatchSize: 0, // should default to 3
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Total != 4 {
		t.Errorf("expected total 4, got %d", result.Total)
	}
	if result.Success != 4 {
		t.Errorf("expected success 4, got %d", result.Success)
	}
}

func TestResultDuration(t *testing.T) {
	d := newMockDeployer()
	apps := makeTestApps(1)
	config := mcp.BatchDeployConfig{
		Apps:     apps,
		Strategy: mcp.StrategySequential,
	}

	result := executeBatchDeploy(context.Background(), d, config)

	if result.Duration < 0 {
		t.Errorf("expected non-negative duration, got %f", result.Duration)
	}
}
