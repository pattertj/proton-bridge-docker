package handlers

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestStateFilePermissions(t *testing.T) {
	data, err := os.ReadFile("state.go")
	if err != nil {
		t.Fatalf("failed to read state.go: %v", err)
	}
	src := string(data)

	if strings.Contains(src, "0644") {
		t.Error("state.go still uses 0644 permissions; should be 0600")
	}
	if !strings.Contains(src, "0600") {
		t.Error("state.go should use 0600 for state file permissions")
	}
}

func TestAddAccount(t *testing.T) {
	s := &AppState{}

	s.AddAccount("alice@proton.me")
	accs := s.GetAccounts()
	if len(accs) != 1 || accs[0] != "alice@proton.me" {
		t.Errorf("after first add: %v", accs)
	}
	if !s.IsSetupComplete() {
		t.Error("setup should be complete after adding account")
	}

	// Duplicate should not add again
	s.AddAccount("alice@proton.me")
	if len(s.GetAccounts()) != 1 {
		t.Error("duplicate add should be ignored")
	}

	// Second account
	s.AddAccount("bob@proton.me")
	if len(s.GetAccounts()) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(s.GetAccounts()))
	}
}

func TestRemoveAccount(t *testing.T) {
	s := &AppState{}
	s.AddAccount("alice@proton.me")
	s.AddAccount("bob@proton.me")

	s.RemoveAccount("alice@proton.me")
	accs := s.GetAccounts()
	if len(accs) != 1 || accs[0] != "bob@proton.me" {
		t.Errorf("after remove alice: %v", accs)
	}
	if !s.IsSetupComplete() {
		t.Error("should still be complete with one account")
	}

	s.RemoveAccount("bob@proton.me")
	if len(s.GetAccounts()) != 0 {
		t.Error("should have 0 accounts")
	}
	if s.IsSetupComplete() {
		t.Error("should not be complete with 0 accounts")
	}

	// Remove non-existent should be safe
	s.RemoveAccount("nobody@proton.me")
	if len(s.GetAccounts()) != 0 {
		t.Error("removing non-existent should not error")
	}
}

func TestGetAccountsReturnsCopy(t *testing.T) {
	s := &AppState{}
	s.AddAccount("user@proton.me")

	accs := s.GetAccounts()
	accs[0] = "modified"

	// Original should be unchanged
	if s.GetAccounts()[0] != "user@proton.me" {
		t.Error("GetAccounts should return a copy, not a reference")
	}
}

func TestStateMarshalRoundtrip(t *testing.T) {
	s := &AppState{
		SetupComplete:     true,
		LastBridgeVersion: "v3.22.0",
		Accounts:          []string{"a@pm.me", "b@pm.me"},
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded AppState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !loaded.SetupComplete {
		t.Error("SetupComplete should be true")
	}
	if loaded.LastBridgeVersion != "v3.22.0" {
		t.Errorf("version = %q", loaded.LastBridgeVersion)
	}
	if len(loaded.Accounts) != 2 {
		t.Errorf("accounts len = %d", len(loaded.Accounts))
	}
}

func TestIsSetupComplete_Empty(t *testing.T) {
	s := &AppState{}
	if s.IsSetupComplete() {
		t.Error("empty state should not be setup complete")
	}
}

func TestAccountDetails(t *testing.T) {
	s := &AppState{}

	// Initially empty
	if len(s.GetAccountDetails()) != 0 {
		t.Error("should start with no account details")
	}

	// Set details
	details := []AccountInfo{
		{Email: "alice@proton.me", Username: "alice@proton.me", Password: "bridge-pass-1", IMAPPort: 1143, SMTPPort: 1025, Security: "STARTTLS"},
		{Email: "bob@proton.me", Username: "bob@proton.me", Password: "bridge-pass-2", IMAPPort: 1143, SMTPPort: 1025, Security: "STARTTLS"},
	}
	s.SetAccountDetails(details)

	got := s.GetAccountDetails()
	if len(got) != 2 {
		t.Fatalf("expected 2 details, got %d", len(got))
	}
	if got[0].Email != "alice@proton.me" || got[0].Password != "bridge-pass-1" {
		t.Errorf("first detail = %+v", got[0])
	}
	if got[1].Username != "bob@proton.me" {
		t.Errorf("second detail username = %q", got[1].Username)
	}
}

func TestAccountDetailsReturnsCopy(t *testing.T) {
	s := &AppState{}
	s.SetAccountDetails([]AccountInfo{{Email: "user@proton.me", Password: "secret"}})

	got := s.GetAccountDetails()
	got[0].Password = "modified"

	if s.GetAccountDetails()[0].Password != "secret" {
		t.Error("GetAccountDetails should return a copy")
	}
}

func TestAccountDetailsMarshalRoundtrip(t *testing.T) {
	s := &AppState{
		SetupComplete: true,
		Accounts:      []string{"user@pm.me"},
		AccountDetails: []AccountInfo{
			{Email: "user@pm.me", Username: "user@pm.me", Password: "bp123", IMAPPort: 1143, SMTPPort: 1025, Security: "STARTTLS", Mode: "combined"},
		},
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded AppState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(loaded.AccountDetails) != 1 {
		t.Fatalf("expected 1 account detail, got %d", len(loaded.AccountDetails))
	}
	d := loaded.AccountDetails[0]
	if d.Email != "user@pm.me" || d.Password != "bp123" || d.Mode != "combined" {
		t.Errorf("roundtrip detail = %+v", d)
	}
}
