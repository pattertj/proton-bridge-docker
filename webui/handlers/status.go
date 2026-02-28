package handlers

import (
	"os/exec"
	"strings"
)

// StatusInfo represents the full status response
type StatusInfo struct {
	BridgeState    string                    `json:"bridgeState"`
	SetupComplete  bool                      `json:"setupComplete"`
	BridgeRunning  bool                      `json:"bridgeRunning"`
	SocatSMTP      bool                      `json:"socatSmtp"`
	SocatIMAP      bool                      `json:"socatImap"`
	Accounts       []string                  `json:"accounts"`
	SyncProgress   map[string]*SyncProgress  `json:"syncProgress,omitempty"`
}

// GetStatusInfo gathers system status
func GetStatusInfo(bm *BridgeManager, state *AppState) StatusInfo {
	info := StatusInfo{
		BridgeState:   bm.GetState().String(),
		SetupComplete: state.IsSetupComplete(),
		BridgeRunning: isBridgeAlive(),
		SocatSMTP:     isSocatAlive("1025"),
		SocatIMAP:     isSocatAlive("1143"),
		Accounts:      state.GetAccounts(),
		SyncProgress:  bm.GetSyncProgress(),
	}
	return info
}

func isBridgeAlive() bool {
	cmd := exec.Command("pgrep", "-f", "bridge")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func isSocatAlive(port string) bool {
	cmd := exec.Command("pgrep", "-f", "socat.*"+port)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}
