package handlers

import (
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

func TestIsSetupComplete_Empty(t *testing.T) {
	s := &AppState{}
	if s.IsSetupComplete() {
		t.Error("empty state should not be setup complete")
	}
}
