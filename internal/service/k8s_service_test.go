package service

import (
	"testing"

	k8sapi "k8s.io/api/core/v1"
)

func TestFirstContainerImage_Empty(t *testing.T) {
	if got := firstContainerImage(nil); got != "" {
		t.Errorf("nil containers: got %q", got)
	}
	if got := firstContainerImage([]k8sapi.Container{}); got != "" {
		t.Errorf("empty containers: got %q", got)
	}
}

func TestFirstContainerImage_UsesFirst(t *testing.T) {
	got := firstContainerImage([]k8sapi.Container{
		{Image: "nginx:1"},
		{Image: "sidecar:1"},
	})
	if got != "nginx:1" {
		t.Errorf("got %q, want nginx:1", got)
	}
}

func TestTotalRestartCount_Empty(t *testing.T) {
	if got := totalRestartCount(nil); got != 0 {
		t.Errorf("nil statuses: got %d", got)
	}
}

func TestTotalRestartCount_SumsAll(t *testing.T) {
	got := totalRestartCount([]k8sapi.ContainerStatus{
		{RestartCount: 2},
		{RestartCount: 3},
	})
	if got != 5 {
		t.Errorf("got %d, want 5", got)
	}
}
