package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetStatusInfo(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	state.AddAccount("test@proton.me")

	info := GetStatusInfo(bm, state)

	if info.BridgeState != "running" {
		t.Errorf("expected bridge state 'running', got %q", info.BridgeState)
	}
	if !info.SetupComplete {
		t.Error("expected setup complete with account added")
	}
	if len(info.Accounts) != 1 || info.Accounts[0] != "test@proton.me" {
		t.Errorf("accounts = %v, want [test@proton.me]", info.Accounts)
	}
	// BridgeRunning/SocatSMTP/SocatIMAP depend on running processes,
	// so we just verify they're boolean (false in test environment)
	if info.BridgeRunning {
		t.Log("bridge appears to be running (unexpected in test)")
	}
}

func TestStatusEndpoint(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	state.AddAccount("user@pm.me")
	api := NewAPI(bm, state)

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	api.Status(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp StatusInfo
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.BridgeState != "running" {
		t.Errorf("bridgeState = %q", resp.BridgeState)
	}
	if !resp.SetupComplete {
		t.Error("setupComplete should be true")
	}
	if len(resp.Accounts) != 1 {
		t.Errorf("expected 1 account, got %d", len(resp.Accounts))
	}
}

func TestGetAccountsWithDetails(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	state.AddAccount("user@proton.me")
	state.SetAccountDetails([]AccountInfo{
		{Email: "user@proton.me", Username: "user@proton.me", Password: "bridge-pass", IMAPPort: 1143, SMTPPort: 1025, Security: "STARTTLS", Mode: "combined"},
	})
	a := NewAPI(bm, state)

	req := httptest.NewRequest("GET", "/api/accounts", nil)
	w := httptest.NewRecorder()
	a.GetAccounts(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	accounts := resp["accounts"].([]any)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	acc := accounts[0].(map[string]any)
	if acc["username"] != "user@proton.me" {
		t.Errorf("username = %v", acc["username"])
	}
	if acc["password"] != "bridge-pass" {
		t.Errorf("password = %v", acc["password"])
	}
	if acc["mode"] != "combined" {
		t.Errorf("mode = %v", acc["mode"])
	}
	// Should NOT have needsRefresh when details are present
	if _, ok := resp["needsRefresh"]; ok {
		t.Error("should not have needsRefresh when details exist")
	}
}

func TestGetAccountsFallbackWithoutDetails(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	state.AddAccount("user@proton.me")
	// No SetAccountDetails — simulates pre-refresh state
	a := NewAPI(bm, state)

	req := httptest.NewRequest("GET", "/api/accounts", nil)
	w := httptest.NewRecorder()
	a.GetAccounts(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	accounts := resp["accounts"].([]any)
	acc := accounts[0].(map[string]any)
	if acc["username"] != "" {
		t.Errorf("username should be empty without details, got %v", acc["username"])
	}
	if resp["needsRefresh"] != true {
		t.Error("should have needsRefresh=true without details")
	}
}

func TestLoginValidation(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	api := NewAPI(bm, state)

	// Empty body
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.Login(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for empty credentials, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "username and password required" {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestLoginTwoFactorValidation(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	api := NewAPI(bm, state)

	req := httptest.NewRequest("POST", "/api/login/2fa", strings.NewReader(`{"code":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.LoginTwoFactor(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for empty 2FA code, got %d", w.Code)
	}
}

func TestToggleModeValidation(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	api := NewAPI(bm, state)

	// Missing ID
	req := httptest.NewRequest("POST", "/api/accounts//mode", nil)
	w := httptest.NewRecorder()
	api.ToggleMode(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for missing account ID, got %d", w.Code)
	}
}

func TestLogoutValidation(t *testing.T) {
	bm := NewBridgeManager()
	state := &AppState{}
	api := NewAPI(bm, state)

	req := httptest.NewRequest("POST", "/api/logout/", nil)
	w := httptest.NewRecorder()
	api.Logout(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for missing account ID, got %d", w.Code)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, 201, map[string]string{"msg": "created"})

	if w.Code != 201 {
		t.Errorf("status = %d, want 201", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["msg"] != "created" {
		t.Errorf("body = %v", resp)
	}
}
