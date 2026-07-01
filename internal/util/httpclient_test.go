package util

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultClient_Timeout(t *testing.T) {
	expectedTimeout := 30 * time.Second
	if DefaultClient.Timeout != expectedTimeout {
		t.Errorf("DefaultClient.Timeout = %v, want %v", DefaultClient.Timeout, expectedTimeout)
	}
}

func TestDefaultClient_Transport(t *testing.T) {
	transport, ok := DefaultClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("DefaultClient.Transport is not *http.Transport")
	}

	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 10", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}
}