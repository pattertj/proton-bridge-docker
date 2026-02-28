package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpointTLSField(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	api := NewAPI(bm, state)

	// Default: TLS disabled
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	api.Health(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["tls"] != false {
		t.Errorf("expected tls=false, got %v", resp["tls"])
	}

	// Enable TLS
	api.SetTLSEnabled(true)
	req = httptest.NewRequest("GET", "/api/health", nil)
	w = httptest.NewRecorder()
	api.Health(w, req)

	resp = map[string]any{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["tls"] != true {
		t.Errorf("expected tls=true, got %v", resp["tls"])
	}
}
