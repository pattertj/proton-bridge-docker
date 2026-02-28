package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// API holds the HTTP handler dependencies
type API struct {
	bridge *BridgeManager
	state  *AppState
}

func NewAPI(bridge *BridgeManager, state *AppState) *API {
	return &API{bridge: bridge, state: state}
}

// Health is a simple healthcheck endpoint
func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"bridge":  isBridgeAlive(),
	})
}

// Status returns full bridge and system status
func (a *API) Status(w http.ResponseWriter, r *http.Request) {
	info := GetStatusInfo(a.bridge, a.state)
	writeJSON(w, http.StatusOK, info)
}

// GetAccounts returns configured accounts with credentials
func (a *API) GetAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.bridge.GetAccounts()
	if err != nil {
		// If we can't enter CLI mode, return what we know from state
		log.Printf("[api] GetAccounts error: %v, returning state-based response", err)
		stateAccounts := a.state.GetAccounts()
		fallback := make([]Account, len(stateAccounts))
		for i, email := range stateAccounts {
			fallback[i] = Account{
				ID:       string(rune('0' + i)),
				Email:    email,
				IMAPPort: 1143,
				SMTPPort: 1025,
				Security: "STARTTLS",
				Status:   "unknown (bridge in non-interactive mode)",
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"accounts": fallback,
			"note":     "Credentials only available during setup mode. Click 'Refresh Credentials' to enter setup mode.",
		})
		// Re-exit setup mode if we entered it
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
