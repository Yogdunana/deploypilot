package util

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultClient_Timeout(t *testing.T) {
	if DefaultClient.Timeout != 30*time.Second {
		t.Errorf("DefaultClient.Timeout = %v, want 30s", DefaultClient.Timeout)
	}
}

func TestDefaultClient_Transport(t *testing.T) {
	transport, ok := DefaultClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("DefaultClient.Transport is not *http.Transport")
	}

	if transport.MaxIdleConns != 100 {
		t.Errorf("Transport.MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("Transport.MaxIdleConnsPerHost = %d, want 10", transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("Transport.IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}
}

func TestDefaultClient_IsNotNil(t *testing.T) {
	if DefaultClient == nil {
		t.Error("DefaultClient should not be nil")
	}
}

func TestDefaultClient_TransportNotNil(t *testing.T) {
	if DefaultClient.Transport == nil {
		t.Error("DefaultClient.Transport should not be nil")
	}
}