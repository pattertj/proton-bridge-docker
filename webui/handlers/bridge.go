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

// bridgeCLI manages a bridge --cli subprocess
type bridgeCLI struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	output chan string
	done   chan struct{}
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

	// Send login command, wait for Username: prompt
	cli.sendRaw("login")
	output, err := cli.waitForMarkers(30*time.Second, []string{"Username:"})
	if err != nil {
		return nil, fmt.Errorf("login command failed: %w", err)
	}
	log.Printf("[bridge] Login prompt output: %s", truncate(output, 200))

	// Send username, wait for Password: prompt
	cli.sendRaw(username)
	output, err = cli.waitForMarkers(30*time.Second, []string{"Password:"})
	if err != nil {
		return nil, fmt.Errorf("username entry failed: %w", err)
	}
	log.Println("[bridge] Username submitted, awaiting password prompt...")

	// Send password, wait for result (2FA prompt, success, or >>> prompt)
	cli.sendRaw(password)
	output, err = cli.waitForMarkers(30*time.Second, []string{
		"Two factor code:", "2FA", "two factor",
		"was added successfully", "already logged in",
		"Incorrect login credentials", "invalid",
		">>> ",
	})
	if err != nil {
		return nil, fmt.Errorf("password entry failed: %w", err)
	}
	log.Println("[bridge] Password submitted, awaiting response...")

	// Check for 2FA prompt
	if strings.Contains(output, "Two factor code:") || strings.Contains(output, "two factor") || strings.Contains(output, "2FA") {
		return &LoginResult{NeedsTwoFA: true, Message: "Two-factor authentication code required"}, nil
	}

	// Check for success
	if strings.Contains(output, "was added successfully") || strings.Contains(output, "already logged in") {
		return &LoginResult{Success: true, Message: "Account added successfully"}, nil
	}

	// Check for auth failure
	if strings.Contains(output, "Incorrect login credentials") || strings.Contains(output, "invalid") {
		return &LoginResult{Error: "Incorrect login credentials"}, nil
	}

	// Wait a bit longer for async responses
	time.Sleep(3 * time.Second)
	extra := cli.drainOutput()
	combined := output + extra

	if strings.Contains(combined, "was added successfully") {
		return &LoginResult{Success: true, Message: "Account added successfully"}, nil
	}
	if strings.Contains(combined, "Two factor code:") || strings.Contains(combined, "two factor") {
		return &LoginResult{NeedsTwoFA: true, Message: "Two-factor authentication code required"}, nil
	}
	if strings.Contains(combined, "Incorrect") || strings.Contains(combined, "failed") {
		return &LoginResult{Error: "Authentication failed"}, nil
	}

	// Ambiguous result — log detail server-side, return generic message
	log.Printf("[bridge] Ambiguous login result: %s", truncate(combined, 300))
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

	cli.sendRaw(code)
	output, err := cli.waitForMarkers(30*time.Second, []string{
		"was added successfully", "already logged in",
		"Incorrect", "invalid", "failed",
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
		return &LoginResult{Success: true, Message: "Account added successfully"}, nil
	}
	if strings.Contains(combined, "Incorrect") || strings.Contains(combined, "invalid") || strings.Contains(combined, "failed") {
		return &LoginResult{Error: "Invalid 2FA code"}, nil
	}

	log.Printf("[bridge] Ambiguous 2FA result: %s", truncate(combined, 300))
	return &LoginResult{Message: "2FA submitted. Please check status for result."}, nil
}

// GetAccounts retrieves account info from the bridge CLI
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

	output, err := cli.execute("info")
	if err != nil {
		return nil, fmt.Errorf("info command failed: %w", err)
	}

	return parseAccounts(output), nil
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
	// Use script to provide a PTY so bridge outputs >>> prompts
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
			c.output <- stripANSI(string(buf[:n]))
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
	if err := c.sendRaw(command); err != nil {
		return "", fmt.Errorf("write command: %w", err)
	}
	return c.waitForPrompt(30 * time.Second)
}

func (c *bridgeCLI) executeRaw(input string) (string, error) {
	if err := c.sendRaw(input); err != nil {
		return "", fmt.Errorf("write input: %w", err)
	}
	return c.waitForPrompt(30 * time.Second)
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
var controlRegex = regexp.MustCompile(`[\x00-\x09\x0b\x0c\x0e-\x1f]`)

func stripANSI(s string) string {
	s = ansiRegex.ReplaceAllString(s, "")
	s = controlRegex.ReplaceAllString(s, "")
	return s
}

var (
	emailRegex    = regexp.MustCompile(`(?i)account:\s*(.+@.+)`)
	usernameRegex = regexp.MustCompile(`(?i)username:\s*(.+)`)
	passwordRegex = regexp.MustCompile(`(?i)password:\s*(.+)`)
	imapRegex     = regexp.MustCompile(`(?i)imap.*?port:\s*(\d+)`)
	smtpRegex     = regexp.MustCompile(`(?i)smtp.*?port:\s*(\d+)`)
	securityRegex = regexp.MustCompile(`(?i)security:\s*(.+)`)
	modeRegex     = regexp.MustCompile(`(?i)mode:\s*(.+)`)
	statusRegex   = regexp.MustCompile(`(?i)status:\s*(.+)`)
)

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
		if strings.Contains(line, "@") && (strings.Contains(strings.ToLower(line), "account") || emailRegex.MatchString(line)) {
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
