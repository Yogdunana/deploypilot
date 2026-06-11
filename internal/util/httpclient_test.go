package util

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultClient_NotNil(t *testing.T) {
	if DefaultClient == nil {
		t.Fatal("DefaultClient is nil, want non-nil *http.Client")
	}
}

func TestDefaultClient_Transport(t *testing.T) {
	tr, ok := DefaultClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("DefaultClient.Transport = %T, want *http.Transport", DefaultClient.Transport)
	}
	if tr.MaxIdleConns <= 0 {
		t.Errorf("MaxIdleConns = %d, want > 0", tr.MaxIdleConns)
	}
	if tr.MaxIdleConnsPerHost <= 0 {
		t.Errorf("MaxIdleConnsPerHost = %d, want > 0", tr.MaxIdleConnsPerHost)
	}
	if tr.IdleConnTimeout <= 0 {
		t.Errorf("IdleConnTimeout = %v, want > 0", tr.IdleConnTimeout)
	}
}

func TestDefaultClient_Timeout(t *testing.T) {
	if DefaultClient.Timeout <= 0 {
		t.Errorf("DefaultClient.Timeout = %v, want > 0", DefaultClient.Timeout)
	}
	if DefaultClient.Timeout > 5*time.Minute {
		t.Errorf("DefaultClient.Timeout = %v, want reasonable (< 5min)", DefaultClient.Timeout)
	}
}
