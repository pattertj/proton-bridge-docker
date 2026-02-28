package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
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

func TestGetAccountsDoesNotRestartBridge(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	api := NewAPI(bm, state)

	// GET /api/accounts should return cached data without changing bridge state
	stateBefore := bm.GetState()
	req := httptest.NewRequest("GET", "/api/accounts", nil)
	w := httptest.NewRecorder()
	api.GetAccounts(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	stateAfter := bm.GetState()
	if stateBefore != stateAfter {
		t.Errorf("bridge state changed from %v to %v during GET /api/accounts", stateBefore, stateAfter)
	}
}

func TestLoginRejectsOversizedBody(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	api := NewAPI(bm, state)

	// Create a body larger than 10KB
	bigBody := strings.Repeat("x", 20*1024)
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(bigBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.Login(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for oversized body, got %d", w.Code)
	}
}
