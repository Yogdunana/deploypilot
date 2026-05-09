package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBTPanelClient_Login(t *testing.T) {
	// Create a test server that simulates BT Panel login
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" && r.Method == http.MethodPost {
			// Set a cookie for authentication
			w.Header().Set("Set-Cookie", "bt_user_token=test-token; Path=/")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": true, "msg": "Login successful"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewBTPanelClient(server.URL, "admin", "password")
	
	// Test login
	err := client.login(context.Background())
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if client.cookie == "" {
		t.Fatal("cookie not set after login")
	}
}

func TestBTPanelClient_OpenFirewall(t *testing.T) {
	// Create a test server that simulates BT Panel API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" && r.Method == http.MethodPost {
			w.Header().Set("Set-Cookie", "bt_user_token=test-token; Path=/")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": true, "msg": "Login successful"}`))
			return
		}

		if r.URL.Path == "/firewall/add" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": true, "msg": "Firewall rule added"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewBTPanelClient(server.URL, "admin", "password")

	// Test opening firewall port
	err := client.OpenFirewall(context.Background(), 8080, "TCP")
	if err != nil {
		t.Fatalf("OpenFirewall failed: %v", err)
	}
}

func TestBTPanelClient_CloseFirewall(t *testing.T) {
	// Create a test server that simulates BT Panel API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" && r.Method == http.MethodPost {
			w.Header().Set("Set-Cookie", "bt_user_token=test-token; Path=/")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": true, "msg": "Login successful"}`))
			return
		}

		if r.URL.Path == "/firewall" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": true,
				"data": {
					"list": [
						{
							"id": "1",
							"port": "8080",
							"protocol": "TCP",
							"ps": "deploypilot"
						}
					]
				}
			}`))
			return
		}

		if r.URL.Path == "/firewall/del" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": true, "msg": "Firewall rule deleted"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewBTPanelClient(server.URL, "admin", "password")

	// Test closing firewall port
	err := client.CloseFirewall(context.Background(), 8080, "TCP")
	if err != nil {
		t.Fatalf("CloseFirewall failed: %v", err)
	}
}

func TestBTPanelClient_CreateReverseProxy(t *testing.T) {
	// Create a test server that simulates BT Panel API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" && r.Method == http.MethodPost {
			w.Header().Set("Set-Cookie", "bt_user_token=test-token; Path=/")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": true, "msg": "Login successful"}`))
			return
		}

		if r.URL.Path == "/site/add" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": true, "msg": "Site added"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewBTPanelClient(server.URL, "admin", "password")

	// Test creating reverse proxy
	err := client.CreateReverseProxy(context.Background(), "test.example.com", "http://localhost:3000", 3000)
	if err != nil {
		t.Fatalf("CreateReverseProxy failed: %v", err)
	}
}
