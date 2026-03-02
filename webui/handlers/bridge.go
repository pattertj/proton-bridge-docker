package handlers

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// BridgeState represents the current operational state of the bridge
type BridgeState int

const (
	StateRunning      BridgeState = iota // Bridge running non-interactively
	StateSetupMode                       // Bridge CLI active for setup
	StateTransitioning                   // Switching between modes
	StateStopped                         // Bridge not running
)

func (s BridgeState) String() string {
	switch s {
	case StateRunning:
		return "running"
	case StateSetupMode:
		return "setup"
	case StateTransitioning:
		return "transitioning"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// Account holds parsed bridge account information
type Account struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	IMAPPort int    `json:"imapPort"`
	SMTPPort int    `json:"smtpPort"`
	Security string `json:"security"`
	Mode     string `json:"mode"`
	Status   string `json:"status"`
}

// LoginResult represents the outcome of a login attempt
type LoginResult struct {
	Success    bool   `json:"success"`
	NeedsTwoFA bool   `json:"needs2fa"`
	Syncing    bool   `json:"syncing,omitempty"`
	Error      string `json:"error,omitempty"`
	Message    string `json:"message,omitempty"`
}

// SyncProgress holds mailbox sync state
type SyncProgress struct {
	Account    string  `json:"account"`
	Percentage float64 `json:"percentage"`
	Syncing    bool    `json:"syncing"`
}

// BridgeManager manages the bridge process and CLI interactions
type BridgeManager struct {
	mu           sync.Mutex
	state        BridgeState
	cli          *bridgeCLI
	syncProgress map[string]*SyncProgress
}

// syncProgressRegex matches bridge sync progress lines like:
//   Sync (pattertj): 5.2% (Elapsed: 0.4s, ETA: 6.8s)
var syncProgressRegex = regexp.MustCompile(`Sync\s*\(([^)]+)\)\s*:\s*([0-9.]+)%`)

// syncBegunRegex matches "A sync has begun for <account>."
var syncBegunRegex = regexp.MustCompile(`sync has begun for (\S+)`)

// syncFinishedRegex matches "A sync has finished for <account>."
var syncFinishedRegex = regexp.MustCompile(`sync has finished for (\S+)`)

// bridgeCLI manages a bridge --cli subprocess
type bridgeCLI struct {
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	reader       *bufio.Reader
	output       chan string
	done         chan struct{}
	syncCallback func(account string, pct float64)
}

func NewBridgeManager() *BridgeManager {
	return &BridgeManager{
		state:        StateRunning, // Assume bridge started by entrypoint
		syncProgress: make(map[string]*SyncProgress),
	}
}

// GetState returns the current bridge state
func (bm *BridgeManager) GetState() BridgeState {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.state
}

// GetSyncProgress returns current sync progress for all accounts
func (bm *BridgeManager) GetSyncProgress() map[string]*SyncProgress {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	result := make(map[string]*SyncProgress)
	for k, v := range bm.syncProgress {
		cp := *v
		result[k] = &cp
	}
	return result
}

// updateSyncProgress updates the sync progress for a given account
func (bm *BridgeManager) updateSyncProgress(account string, pct float64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.syncProgress[account] = &SyncProgress{
		Account:    account,
		Percentage: pct,
		Syncing:    pct < 100,
	}
	log.Printf("[bridge] Sync progress for %s: %.1f%%", account, pct)
}

// EnterSetupMode stops the non-interactive bridge and starts CLI mode
func (bm *BridgeManager) EnterSetupMode() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.state == StateSetupMode && bm.cli != nil {
		return nil // Already in setup mode
	}

	bm.state = StateTransitioning
	log.Println("[bridge] Entering setup mode...")

	// Kill the non-interactive bridge
	if err := killBridgeProcess(); err != nil {
		log.Printf("[bridge] Warning: could not kill bridge process: %v", err)
	}

	// Wait for process to die
	time.Sleep(2 * time.Second)

	// Start CLI mode
	cli, err := newBridgeCLI()
	if err != nil {
		bm.state = StateStopped
		return fmt.Errorf("failed to start bridge CLI: %w", err)
	}

	cli.syncCallback = func(account string, pct float64) {
		bm.updateSyncProgress(account, pct)
	}
	bm.cli = cli
	bm.state = StateSetupMode
	log.Println("[bridge] Setup mode active")
	return nil
}

// ExitSetupMode closes the CLI and restarts non-interactive bridge
func (bm *BridgeManager) ExitSetupMode() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.state != StateSetupMode {
		return nil
	}

	bm.state = StateTransitioning
	log.Println("[bridge] Exiting setup mode...")

	if bm.cli != nil {
		bm.cli.close()
		bm.cli = nil
	}

	// Wait for CLI to fully exit
	time.Sleep(2 * time.Second)

	// Restart non-interactive bridge
	if err := startBridgeNonInteractive(); err != nil {
		bm.state = StateStopped
		return fmt.Errorf("failed to restart bridge: %w", err)
	}

	bm.state = StateRunning
	log.Println("[bridge] Bridge restarted in non-interactive mode")
	return nil
}

// Login performs the multi-step login flow
func (bm *BridgeManager) Login(username, password string) (*LoginResult, error) {
	if err := bm.EnterSetupMode(); err != nil {
		return nil, err
	}

	bm.mu.Lock()
	cli := bm.cli
	bm.mu.Unlock()

	if cli == nil {
		return nil, fmt.Errorf("CLI not available")
	}

	// Send login command. The bridge may respond with "Username:" for a new
	// login, or it may output sync progress / "already logged in" if an
	// account is already active.
	output, err := cli.sendAndWait("login", 30*time.Second, []string{
		"Username:",
		"already logged in",
		"sync has begun", "Sync",
		">>> ",
	})
	if err != nil {
		return nil, fmt.Errorf("login command failed: %w", err)
	}
	log.Printf("[bridge] Login prompt output: %s", truncate(output, 200))

	// If account is already logged in or syncing, no need to continue login flow
	lowerOutput := strings.ToLower(output)
	if strings.Contains(lowerOutput, "already logged in") || strings.Contains(lowerOutput, "sync has begun") || strings.Contains(lowerOutput, "sync (") {
		syncing := strings.Contains(lowerOutput, "sync has begun") || strings.Contains(lowerOutput, "sync (")
		log.Println("[bridge] Login result: account already active/syncing")
		return &LoginResult{Success: true, Syncing: syncing, Message: "Account already logged in"}, nil
	}

	// If we got a prompt back without "Username:", the account may already be set up
	if !strings.Contains(output, "Username:") {
		log.Printf("[bridge] Login command returned without Username prompt: %s", truncate(output, 300))
		return &LoginResult{Success: true, Message: "Account already configured"}, nil
	}

	// Send username, wait for Password: prompt
	output, err = cli.sendAndWait(username, 30*time.Second, []string{"Password:"})
	if err != nil {
		return nil, fmt.Errorf("username entry failed: %w", err)
	}
	log.Println("[bridge] Username submitted, awaiting password prompt...")

	// Send password, wait for result (2FA prompt, success, or >>> prompt).
	// Drain echo first to avoid matching ">>> " in PTY readline echo.
	log.Println("[bridge] Password submitted, awaiting response...")
	output, err = cli.sendAndWait(password, 30*time.Second, []string{
		"Two factor code:", "2FA", "two factor",
		"was added successfully", "already logged in",
		"Incorrect login credentials", "invalid",
		"sync has begun", "Sync",
		">>> ",
	})
	if err != nil {
		return nil, fmt.Errorf("password entry failed: %w", err)
	}
	log.Printf("[bridge] Password response raw: %s", truncate(output, 500))

	// Check for 2FA prompt
	if strings.Contains(output, "Two factor code:") || strings.Contains(output, "two factor") || strings.Contains(output, "2FA") {
		log.Println("[bridge] Login result: needs 2FA")
		return &LoginResult{NeedsTwoFA: true, Message: "Two-factor authentication code required"}, nil
	}

	// Check for success (including sync starting, which means login worked)
	if strings.Contains(output, "was added successfully") || strings.Contains(output, "already logged in") ||
		strings.Contains(output, "sync has begun") || strings.Contains(output, "Sync (") {
		syncing := strings.Contains(output, "sync has begun") || strings.Contains(output, "Sync (")
		if syncing {
			// Immediately populate syncProgress so the frontend's first poll sees it
			if m := syncBegunRegex.FindStringSubmatch(strings.ToLower(output)); len(m) > 1 {
				account := strings.TrimSuffix(m[1], ".")
				bm.updateSyncProgress(account, 0)
			} else {
				bm.updateSyncProgress(username, 0)
			}
		}
		log.Println("[bridge] Login result: success")
		return &LoginResult{Success: true, Syncing: syncing, Message: "Account added successfully"}, nil
	}

	// Check for auth failure
	if strings.Contains(output, "Incorrect login credentials") || strings.Contains(output, "invalid") {
		log.Println("[bridge] Login result: auth failure")
		return &LoginResult{Error: "Incorrect login credentials"}, nil
	}

	// Wait a bit longer for async responses (bridge authenticates with Proton servers)
	log.Println("[bridge] No immediate result, waiting 8s for async response...")
	time.Sleep(8 * time.Second)
	extra := cli.drainOutput()
	combined := output + extra
	log.Printf("[bridge] Combined output after drain: %s", truncate(combined, 500))

	if strings.Contains(combined, "was added successfully") {
		log.Println("[bridge] Login result (delayed): success")
		return &LoginResult{Success: true, Message: "Account added successfully"}, nil
	}
	if strings.Contains(combined, "Two factor code:") || strings.Contains(combined, "two factor") {
		log.Println("[bridge] Login result (delayed): needs 2FA")
		return &LoginResult{NeedsTwoFA: true, Message: "Two-factor authentication code required"}, nil
	}
	if strings.Contains(combined, "Incorrect") || strings.Contains(combined, "failed") {
		log.Println("[bridge] Login result (delayed): auth failure")
		return &LoginResult{Error: "Authentication failed"}, nil
	}

	// Ambiguous result — log detail server-side, return generic message
	log.Printf("[bridge] Ambiguous login result: %s", truncate(combined, 500))
	return &LoginResult{Message: "Login submitted. Please check status for result."}, nil
}

// SubmitTwoFactor sends the 2FA code
func (bm *BridgeManager) SubmitTwoFactor(code string) (*LoginResult, error) {
	bm.mu.Lock()
	cli := bm.cli
	bm.mu.Unlock()

	if cli == nil || bm.GetState() != StateSetupMode {
		return nil, fmt.Errorf("not in setup mode")
	}

	output, err := cli.sendAndWait(code, 30*time.Second, []string{
		"was added successfully", "already logged in",
		"Incorrect", "invalid", "failed",
		"sync has begun", "Sync",
		">>> ",
	})
	if err != nil {
		return nil, fmt.Errorf("2FA submission failed: %w", err)
	}

	// Drain any extra async output
	time.Sleep(2 * time.Second)
	extra := cli.drainOutput()
	combined := output + extra

	if strings.Contains(combined, "was added successfully") || strings.Contains(combined, "already logged in") {
		syncing := strings.Contains(combined, "sync has begun") || strings.Contains(combined, "Sync (")
		return &LoginResult{Success: true, Syncing: syncing, Message: "Account added successfully"}, nil
	}
	if strings.Contains(combined, "Incorrect") || strings.Contains(combined, "invalid") || strings.Contains(combined, "failed") {
		return &LoginResult{Error: "Invalid 2FA code"}, nil
	}

	log.Printf("[bridge] Ambiguous 2FA result: %s", truncate(combined, 300))
	return &LoginResult{Message: "2FA submitted. Please check status for result."}, nil
}

// GetAccounts retrieves account info from the bridge CLI using a two-step
// approach: first `list` to discover accounts, then `info` + account index
// for each account to get credentials (since `info` is interactive).
func (bm *BridgeManager) GetAccounts() ([]Account, error) {
	if err := bm.EnterSetupMode(); err != nil {
		return nil, err
	}

	bm.mu.Lock()
	cli := bm.cli
	bm.mu.Unlock()

	if cli == nil {
		return nil, fmt.Errorf("CLI not available")
	}

	// Drain any buffered output (e.g. sync messages) before running commands
	cli.drainOutput()

	// Step 1: Get account list
	listOutput, err := cli.execute("list")
	if err != nil {
		return nil, fmt.Errorf("list command failed: %w", err)
	}
	log.Printf("[bridge] list output: %s", truncate(listOutput, 500))

	entries := parseListOutput(listOutput)
	if len(entries) == 0 {
		log.Println("[bridge] no accounts found in list output")
		return nil, nil
	}

	// Step 2: For each account, run `info` and select the account index.
	// The `info` command is interactive — it prompts for an account selection.
	// With a single account, it may auto-select or still prompt.
	var accounts []Account
	for _, entry := range entries {
		// Skip info command for locked (syncing) accounts — it will fail/timeout
		if strings.ToLower(entry.Status) == "locked" {
			log.Printf("[bridge] Skipping info for locked account %s (syncing)", entry.Email)
			accounts = append(accounts, Account{
				ID:       entry.Index,
				Email:    entry.Email,
				Status:   "syncing",
				Mode:     entry.Mode,
				Security: "STARTTLS",
				IMAPPort: 1143,
				SMTPPort: 1025,
			})
			continue
		}

		// Send info command — execute() uses echo-aware prompt detection.
		output, err := cli.execute("info")
		if err != nil {
			log.Printf("[bridge] info command failed for %s: %v (output: %s)", entry.Email, err, truncate(output, 300))
			// Check if it's waiting for account selection
			if strings.Contains(output, "account") || strings.Contains(output, "Account") {
				infoOutput, err2 := cli.sendAndWait(entry.Index, 15*time.Second, []string{">>> "})
				if err2 != nil {
					log.Printf("[bridge] info response failed for %s: %v", entry.Email, err2)
					continue
				}
				log.Printf("[bridge] info output for %s: %s", entry.Email, truncate(infoOutput, 500))
				acct := parseInfoBlock(infoOutput, entry)
				accounts = append(accounts, acct)
				continue
			}
			// Timed out with no useful output — still include account from list
			acct := parseInfoBlock(output, entry)
			accounts = append(accounts, acct)
			continue
		}

		log.Printf("[bridge] info output for %s: %s", entry.Email, truncate(output, 500))

		// Check if info returned directly with credentials (single account auto-select)
		if strings.Contains(output, "Configuration for") || strings.Contains(output, "Username:") {
			acct := parseInfoBlock(output, entry)
			accounts = append(accounts, acct)
			continue
		}

		// If output contains an account selection prompt, send the index
		if strings.Contains(output, "account") || strings.Contains(output, "Account") {
			infoOutput, err := cli.sendAndWait(entry.Index, 15*time.Second, []string{">>> "})
			if err != nil {
				log.Printf("[bridge] info response failed for %s: %v", entry.Email, err)
				continue
			}
			log.Printf("[bridge] info detail for %s: %s", entry.Email, truncate(infoOutput, 500))
			acct := parseInfoBlock(infoOutput, entry)
			accounts = append(accounts, acct)
			continue
		}

		// Fallback: always include the account from list even if info
		// didn't return credentials (e.g. account is locked).
		acct := parseInfoBlock(output, entry)
		accounts = append(accounts, acct)
	}

	return accounts, nil
}

// ToggleMode toggles combined/split address mode for an account
func (bm *BridgeManager) ToggleMode(accountID string) (string, error) {
	safeID, err := sanitizeAccountID(accountID)
	if err != nil {
		return "", err
	}

	if err := bm.EnterSetupMode(); err != nil {
		return "", err
	}

	bm.mu.Lock()
	cli := bm.cli
	bm.mu.Unlock()

	if cli == nil {
		return "", fmt.Errorf("CLI not available")
	}

	output, err := cli.execute("change mode " + safeID)
	if err != nil {
		return "", err
	}
	return output, nil
}

// LogoutAccount removes an account
func (bm *BridgeManager) LogoutAccount(accountID string) (string, error) {
	safeID, err := sanitizeAccountID(accountID)
	if err != nil {
		return "", err
	}

	if err := bm.EnterSetupMode(); err != nil {
		return "", err
	}

	bm.mu.Lock()
	cli := bm.cli
	bm.mu.Unlock()

	if cli == nil {
		return "", fmt.Errorf("CLI not available")
	}

	output, err := cli.execute("logout " + safeID)
	if err != nil {
		return "", err
	}
	return output, nil
}

// IsBridgeProcessRunning checks if a bridge process is alive
func IsBridgeProcessRunning() bool {
	cmd := exec.Command("pgrep", "-f", "bridge.*--noninteractive")
	return cmd.Run() == nil
}

// Close shuts down the bridge manager
func (bm *BridgeManager) Close() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.cli != nil {
		bm.cli.close()
		bm.cli = nil
	}
}

// --- bridgeCLI implementation ---

func newBridgeCLI() (*bridgeCLI, error) {
	// Use script to provide a PTY so bridge outputs >>> prompts.
	cmd := exec.Command("script", "-qc", "/usr/bin/bridge --cli", "/dev/null")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	// Merge stderr into stdout
	cmd.Stderr = cmd.Stdout

	cli := &bridgeCLI{
		cmd:    cmd,
		stdin:  stdin,
		reader: bufio.NewReader(stdout),
		output: make(chan string, 100),
		done:   make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start bridge CLI: %w", err)
	}

	// Start reading stdout in background
	go cli.readLoop()

	// Wait for initial prompt
	if _, err := cli.waitForPrompt(30 * time.Second); err != nil {
		cli.close()
		return nil, fmt.Errorf("waiting for initial prompt: %w", err)
	}

	log.Println("[bridge-cli] CLI ready")
	return cli, nil
}

func (c *bridgeCLI) readLoop() {
	defer close(c.done)
	buf := make([]byte, 4096)
	for {
		n, err := c.reader.Read(buf)
		if n > 0 {
			text := stripANSI(string(buf[:n]))
			c.output <- text

			// Check for sync progress lines and invoke callback
			if c.syncCallback != nil {
				lowerText := strings.ToLower(text)

				// Check for percentage progress: "Sync (user): 5.2%"
				if matches := syncProgressRegex.FindAllStringSubmatch(text, -1); len(matches) > 0 {
					for _, m := range matches {
						var pct float64
						fmt.Sscanf(m[2], "%f", &pct)
						c.syncCallback(m[1], pct)
					}
				}

				// Check for "sync has begun for <account>" → set 0%
				if m := syncBegunRegex.FindStringSubmatch(lowerText); len(m) > 1 {
					account := strings.TrimSuffix(m[1], ".")
					c.syncCallback(account, 0)
				}

				// Check for "sync has finished for <account>" → set 100%
				if m := syncFinishedRegex.FindStringSubmatch(lowerText); len(m) > 1 {
					account := strings.TrimSuffix(m[1], ".")
					c.syncCallback(account, 100)
				}
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("[bridge-cli] Read error: %v", err)
			}
			return
		}
	}
}

func (c *bridgeCLI) waitForPrompt(timeout time.Duration) (string, error) {
	return c.waitForMarkers(timeout, []string{">>> ", "Username:", "Password:", "Two factor code:"})
}

func (c *bridgeCLI) waitForMarkers(timeout time.Duration, markers []string) (string, error) {
	var accumulated strings.Builder
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case chunk, ok := <-c.output:
			if !ok {
				return accumulated.String(), fmt.Errorf("CLI process ended")
			}
			accumulated.WriteString(chunk)
			s := accumulated.String()
			for _, marker := range markers {
				if strings.Contains(s, marker) {
					return s, nil
				}
			}
		case <-timer.C:
			return accumulated.String(), fmt.Errorf("timeout waiting for prompt (got: %s)", truncate(accumulated.String(), 200))
		}
	}
}

func (c *bridgeCLI) sendRaw(input string) error {
	_, err := fmt.Fprintf(c.stdin, "%s\n", input)
	return err
}

func (c *bridgeCLI) execute(command string) (string, error) {
	return c.sendAndWait(command, 30*time.Second, []string{">>> ", "Username:", "Password:", "Two factor code:"})
}

func (c *bridgeCLI) drainOutput() string {
	var buf strings.Builder
	for {
		select {
		case chunk, ok := <-c.output:
			if !ok {
				return buf.String()
			}
			buf.WriteString(chunk)
		default:
			return buf.String()
		}
	}
}

// sendAndWait sends input then waits for markers, stripping PTY readline echo
// from accumulated output before checking. This avoids the unreliable timing-based
// drain approach — echo and response often arrive together through the PTY.
func (c *bridgeCLI) sendAndWait(input string, timeout time.Duration, markers []string) (string, error) {
	if err := c.sendRaw(input); err != nil {
		return "", fmt.Errorf("write input: %w", err)
	}

	var accumulated strings.Builder
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case chunk, ok := <-c.output:
			if !ok {
				return stripEchoFragments(accumulated.String()), fmt.Errorf("CLI process ended")
			}
			accumulated.WriteString(chunk)
			stripped := stripEchoFragments(accumulated.String())
			for _, marker := range markers {
				if strings.Contains(stripped, marker) {
					log.Printf("[bridge-cli] sendAndWait matched %q in: %s", marker, truncate(stripped, 300))
					return stripped, nil
				}
			}
		case <-timer.C:
			stripped := stripEchoFragments(accumulated.String())
			return stripped, fmt.Errorf("timeout waiting for prompt (got: %s)", truncate(stripped, 200))
		}
	}
}

func (c *bridgeCLI) close() {
	log.Println("[bridge-cli] Closing CLI")
	fmt.Fprintf(c.stdin, "exit\n")
	c.stdin.Close()

	done := make(chan struct{})
	go func() {
		c.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Println("[bridge-cli] Force killing CLI process")
		c.cmd.Process.Kill()
	}
}

// --- Helper functions ---

func killBridgeProcess() error {
	cmd := exec.Command("pkill", "-f", "bridge --noninteractive")
	return cmd.Run()
}

func startBridgeNonInteractive() error {
	cmd := exec.Command("/usr/bin/bridge", "--noninteractive")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

var accountIDRegex = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)

// sanitizeAccountID validates that an account ID contains only safe characters.
func sanitizeAccountID(id string) (string, error) {
	if !accountIDRegex.MatchString(id) {
		return "", fmt.Errorf("invalid account ID: contains disallowed characters")
	}
	return id, nil
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
var controlRegex = regexp.MustCompile(`[\x00-\x09\x0b-\x1f]`)

// echoFragRegex matches PTY readline echo fragments: ">>> " followed by at least
// one non-whitespace character (partial command text like ">>> l", ">>> li", ">>> list").
// It does NOT match the real ">>> " prompt which has only whitespace/nothing after it.
var echoFragRegex = regexp.MustCompile(`>>>\s+\S+`)

func stripANSI(s string) string {
	s = ansiRegex.ReplaceAllString(s, "")
	s = controlRegex.ReplaceAllString(s, "")
	return s
}

var (
	emailRegex    = regexp.MustCompile(`(?i)account:\s*(.+@.+)`)
	configRegex   = regexp.MustCompile(`(?i)configuration for\s+(.+@.+)`)
	usernameRegex = regexp.MustCompile(`(?i)username:\s*(.+)`)
	passwordRegex = regexp.MustCompile(`(?i)password:\s*(.+)`)
	imapRegex     = regexp.MustCompile(`(?i)imap.*?port:\s*(\d+)`)
	smtpRegex     = regexp.MustCompile(`(?i)smtp.*?port:\s*(\d+)`)
	securityRegex = regexp.MustCompile(`(?i)security:\s*(.+)`)
	modeRegex     = regexp.MustCompile(`(?i)mode:\s*(.+)`)
	statusRegex   = regexp.MustCompile(`(?i)status:\s*(.+)`)
	// Matches both formats:
	//   "0:  user@proton.me  connected  combined"        (docs)
	//   "0 : pattertj        (locked   , combined    )"  (actual PTY output)
	listLineRegex = regexp.MustCompile(`(?m)(\d+)\s*:\s+(\S+)\s+.*?(\w+)\s*,\s*(\w+)`)
	// Fallback for non-parenthesized format
	listLineSimpleRegex = regexp.MustCompile(`(?m)^(\d+):\s+(\S+)\s+(\S+)\s+(\S+)`)
)

// listEntry holds the index and email parsed from `list` output
type listEntry struct {
	Index  string
	Email  string
	Status string
	Mode   string
}

// parseListOutput extracts account indices and emails from the `list` command.
// Actual PTY format: "0 : pattertj  (locked, combined)"
// Doc format: "0:  user@proton.me  connected  combined"
func parseListOutput(output string) []listEntry {
	// Strip PTY echo noise like ">>> l >>> li >>> lis >>> list"
	output = stripEcho(output)

	// Try parenthesized format first (actual bridge output)
	matches := listLineRegex.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		// Fall back to simple space-separated format
		matches = listLineSimpleRegex.FindAllStringSubmatch(output, -1)
	}

	var entries []listEntry
	for _, m := range matches {
		entries = append(entries, listEntry{
			Index:  m[1],
			Email:  m[2], // May be truncated username, not full email
			Status: m[3],
			Mode:   m[4],
		})
	}
	return entries
}

// stripEcho removes PTY echo artifacts like ">>> l >>> li >>> lis >>> list"
func stripEcho(s string) string {
	echoRegex := regexp.MustCompile(`>>>\s*\S*\s*`)
	return echoRegex.ReplaceAllString(s, "")
}

// stripEchoFragments removes PTY readline echo fragments (">>> l", ">>> list")
// while preserving the real ">>> " prompt (which has no command text after it).
func stripEchoFragments(s string) string {
	return echoFragRegex.ReplaceAllString(s, "")
}

// parseInfoBlock parses the `info` output for a single account and merges
// with list metadata. The info output has a "Configuration for <email>" header.
func parseInfoBlock(output string, entry listEntry) Account {
	acct := Account{
		ID:       entry.Index,
		Email:    entry.Email,
		Status:   entry.Status,
		Mode:     entry.Mode,
		Security: "STARTTLS",
		IMAPPort: 1143,
		SMTPPort: 1025,
	}

	if m := usernameRegex.FindStringSubmatch(output); len(m) > 1 {
		acct.Username = strings.TrimSpace(m[1])
	}
	if m := passwordRegex.FindStringSubmatch(output); len(m) > 1 {
		acct.Password = strings.TrimSpace(m[1])
	}
	if m := imapRegex.FindStringSubmatch(output); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &acct.IMAPPort)
	}
	if m := smtpRegex.FindStringSubmatch(output); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &acct.SMTPPort)
	}
	if m := securityRegex.FindStringSubmatch(output); len(m) > 1 {
		acct.Security = strings.TrimSpace(m[1])
	}
	// If email wasn't in list, try to extract from info header
	if acct.Email == "" {
		if m := configRegex.FindStringSubmatch(output); len(m) > 1 {
			acct.Email = strings.TrimSpace(m[1])
		}
	}
	// Fall back to username as email if needed
	if acct.Email == "" && acct.Username != "" {
		acct.Email = acct.Username
	}

	return acct
}

func parseAccounts(output string) []Account {
	var accounts []Account

	// Split by account blocks — look for email addresses as delimiters
	blocks := splitAccountBlocks(output)

	for i, block := range blocks {
		account := Account{
			ID:       fmt.Sprintf("%d", i),
			Security: "STARTTLS",
			IMAPPort: 1143,
			SMTPPort: 1025,
		}

		if m := emailRegex.FindStringSubmatch(block); len(m) > 1 {
			account.Email = strings.TrimSpace(m[1])
		} else if m := configRegex.FindStringSubmatch(block); len(m) > 1 {
			account.Email = strings.TrimSpace(m[1])
		}
		if m := usernameRegex.FindStringSubmatch(block); len(m) > 1 {
			account.Username = strings.TrimSpace(m[1])
		}
		if m := passwordRegex.FindStringSubmatch(block); len(m) > 1 {
			account.Password = strings.TrimSpace(m[1])
		}
		if m := imapRegex.FindStringSubmatch(block); len(m) > 1 {
			fmt.Sscanf(m[1], "%d", &account.IMAPPort)
		}
		if m := smtpRegex.FindStringSubmatch(block); len(m) > 1 {
			fmt.Sscanf(m[1], "%d", &account.SMTPPort)
		}
		if m := securityRegex.FindStringSubmatch(block); len(m) > 1 {
			account.Security = strings.TrimSpace(m[1])
		}
		if m := modeRegex.FindStringSubmatch(block); len(m) > 1 {
			account.Mode = strings.TrimSpace(m[1])
		}
		if m := statusRegex.FindStringSubmatch(block); len(m) > 1 {
			account.Status = strings.TrimSpace(m[1])
		}

		if account.Email != "" {
			accounts = append(accounts, account)
		}
	}

	return accounts
}

func splitAccountBlocks(output string) []string {
	// Split on lines that look like account headers (contain @ sign)
	lines := strings.Split(output, "\n")
	var blocks []string
	var current strings.Builder
	inBlock := false

	for _, line := range lines {
		isHeader := strings.Contains(line, "@") && (strings.Contains(strings.ToLower(line), "account") || emailRegex.MatchString(line))
		if !isHeader {
			isHeader = configRegex.MatchString(line)
		}
		if isHeader {
			if inBlock && current.Len() > 0 {
				blocks = append(blocks, current.String())
				current.Reset()
			}
			inBlock = true
		}
		if inBlock {
			current.WriteString(line)
			current.WriteString("\n")
		}
	}
	if current.Len() > 0 {
		blocks = append(blocks, current.String())
	}

	// If no blocks found, treat entire output as one block
	if len(blocks) == 0 && strings.Contains(output, "@") {
		blocks = append(blocks, output)
	}

	return blocks
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
