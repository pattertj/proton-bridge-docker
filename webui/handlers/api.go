package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// API holds the HTTP handler dependencies
type API struct {
	bridge     *BridgeManager
	state      *AppState
	tlsEnabled bool
}

func NewAPI(bridge *BridgeManager, state *AppState) *API {
	return &API{bridge: bridge, state: state}
}

// SetTLSEnabled records whether the server is using TLS
func (a *API) SetTLSEnabled(enabled bool) {
	a.tlsEnabled = enabled
}

// Health is a simple healthcheck endpoint
func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"bridge": isBridgeAlive(),
		"tls":    a.tlsEnabled,
	})
}

// Status returns full bridge and system status
func (a *API) Status(w http.ResponseWriter, r *http.Request) {
	info := GetStatusInfo(a.bridge, a.state)
	writeJSON(w, http.StatusOK, info)
}

// GetAccounts returns cached account info from state (no bridge restart).
func (a *API) GetAccounts(w http.ResponseWriter, r *http.Request) {
	stateAccounts := a.state.GetAccounts()
	accounts := make([]Account, len(stateAccounts))
	for i, email := range stateAccounts {
		accounts[i] = Account{
			ID:       string(rune('0' + i)),
			Email:    email,
			IMAPPort: 1143,
			SMTPPort: 1025,
			Security: "STARTTLS",
			Status:   "cached",
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accounts": accounts,
		"note":     "Use POST /api/accounts/refresh to get live credentials (causes brief bridge restart).",
	})
}

// RefreshAccounts enters setup mode to get live account credentials.
func (a *API) RefreshAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.bridge.GetAccounts()
	if err != nil {
		log.Printf("[api] RefreshAccounts error: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Could not enter setup mode to refresh credentials",
		})
		go a.bridge.ExitSetupMode()
		return
	}

	// Update state with discovered accounts
	for _, acc := range accounts {
		if acc.Email != "" {
			a.state.AddAccount(acc.Email)
		}
	}
	a.state.Save()

	// Exit setup mode after getting info
	go a.bridge.ExitSetupMode()

	writeJSON(w, http.StatusOK, map[string]any{"accounts": accounts})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login initiates the login flow
func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024) // 10KB max
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	result, err := a.bridge.Login(req.Username, req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if result.Success {
		a.state.AddAccount(req.Username)
		a.state.Save()
		// Exit setup mode after successful login
		go a.bridge.ExitSetupMode()
	}

	writeJSON(w, http.StatusOK, result)
}

type twoFactorRequest struct {
	Code string `json:"code"`
}

// LoginTwoFactor submits the 2FA code
func (a *API) LoginTwoFactor(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024) // 10KB max
	var req twoFactorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "2FA code required"})
		return
	}

	result, err := a.bridge.SubmitTwoFactor(req.Code)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if result.Success {
		a.state.Save()
		go a.bridge.ExitSetupMode()
	}

	writeJSON(w, http.StatusOK, result)
}

// ToggleMode switches combined/split address mode
func (a *API) ToggleMode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account id required"})
		return
	}

	output, err := a.bridge.ToggleMode(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	go a.bridge.ExitSetupMode()
	writeJSON(w, http.StatusOK, map[string]string{"message": "Mode toggled", "output": output})
}

// Logout removes an account
func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account id required"})
		return
	}

	output, err := a.bridge.LogoutAccount(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	go a.bridge.ExitSetupMode()
	writeJSON(w, http.StatusOK, map[string]string{"message": "Account removed", "output": output})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
