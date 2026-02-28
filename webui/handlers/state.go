package handlers

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

const stateFilePath = "/root/.webui-state.json"

// AccountInfo stores persisted account credentials
type AccountInfo struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	IMAPPort int    `json:"imap_port"`
	SMTPPort int    `json:"smtp_port"`
	Security string `json:"security"`
	Mode     string `json:"mode,omitempty"`
	Status   string `json:"status,omitempty"`
}

// AppState tracks persistent state across container restarts
type AppState struct {
	mu                sync.RWMutex
	SetupComplete     bool          `json:"setup_complete"`
	LastBridgeVersion string        `json:"last_bridge_version"`
	Accounts          []string      `json:"accounts"`
	AccountDetails    []AccountInfo `json:"account_details,omitempty"`
}

// LoadState reads state from disk or returns defaults
func LoadState() *AppState {
	state := &AppState{}

	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		log.Printf("[state] No existing state file, starting fresh")
		return state
	}

	if err := json.Unmarshal(data, state); err != nil {
		log.Printf("[state] Failed to parse state file: %v", err)
		return &AppState{}
	}

	log.Printf("[state] Loaded state: setup_complete=%v, accounts=%v", state.SetupComplete, state.Accounts)
	return state
}

// Save persists state to disk
func (s *AppState) Save() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Printf("[state] Failed to marshal state: %v", err)
		return
	}

	if err := os.WriteFile(stateFilePath, data, 0600); err != nil {
		log.Printf("[state] Failed to write state file: %v", err)
	}
}

// AddAccount adds an account email to state
func (s *AppState) AddAccount(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, a := range s.Accounts {
		if a == email {
			return
		}
	}
	s.Accounts = append(s.Accounts, email)
	s.SetupComplete = true
}

// RemoveAccount removes an account email from state
func (s *AppState) RemoveAccount(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]string, 0, len(s.Accounts))
	for _, a := range s.Accounts {
		if a != email {
			filtered = append(filtered, a)
		}
	}
	s.Accounts = filtered
	s.SetupComplete = len(filtered) > 0
}

// IsSetupComplete returns whether setup has been completed
func (s *AppState) IsSetupComplete() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SetupComplete
}

// GetAccounts returns the list of account emails
func (s *AppState) GetAccounts() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, len(s.Accounts))
	copy(result, s.Accounts)
	return result
}

// SetAccountDetails replaces all stored account details
func (s *AppState) SetAccountDetails(details []AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AccountDetails = make([]AccountInfo, len(details))
	copy(s.AccountDetails, details)
}

// GetAccountDetails returns a copy of stored account details
func (s *AppState) GetAccountDetails() []AccountInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]AccountInfo, len(s.AccountDetails))
	copy(result, s.AccountDetails)
	return result
}
