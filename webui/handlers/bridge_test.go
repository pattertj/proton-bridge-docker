package handlers

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseAccounts_SingleAccount(t *testing.T) {
	output := `Account: user@proton.me
Username: user@proton.me
Password: abc123secret
IMAP port: 1143
SMTP port: 1025
Security: STARTTLS
Mode: combined
Status: connected

>>> `

	accounts := parseAccounts(output)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	a := accounts[0]
	if a.Email != "user@proton.me" {
		t.Errorf("email = %q, want %q", a.Email, "user@proton.me")
	}
	if a.Username != "user@proton.me" {
		t.Errorf("username = %q, want %q", a.Username, "user@proton.me")
	}
	if a.Password != "abc123secret" {
		t.Errorf("password = %q, want %q", a.Password, "abc123secret")
	}
	if a.IMAPPort != 1143 {
		t.Errorf("imapPort = %d, want 1143", a.IMAPPort)
	}
	if a.SMTPPort != 1025 {
		t.Errorf("smtpPort = %d, want 1025", a.SMTPPort)
	}
	if a.Security != "STARTTLS" {
		t.Errorf("security = %q, want STARTTLS", a.Security)
	}
	if a.Mode != "combined" {
		t.Errorf("mode = %q, want combined", a.Mode)
	}
	if a.Status != "connected" {
		t.Errorf("status = %q, want connected", a.Status)
	}
}

func TestParseAccounts_MultipleAccounts(t *testing.T) {
	output := `Account: alice@proton.me
Username: alice@proton.me
Password: pass1
IMAP port: 1143
SMTP port: 1025
Security: STARTTLS

Account: bob@proton.me
Username: bob@proton.me
Password: pass2
IMAP port: 1143
SMTP port: 1025
Security: STARTTLS

>>> `

	accounts := parseAccounts(output)
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}
	if accounts[0].Email != "alice@proton.me" {
		t.Errorf("first account email = %q", accounts[0].Email)
	}
	if accounts[1].Email != "bob@proton.me" {
		t.Errorf("second account email = %q", accounts[1].Email)
	}
	if accounts[0].Password != "pass1" {
		t.Errorf("first password = %q", accounts[0].Password)
	}
	if accounts[1].Password != "pass2" {
		t.Errorf("second password = %q", accounts[1].Password)
	}
}

func TestParseAccounts_ConfigurationForHeader(t *testing.T) {
	// This is the actual format from `info` command output
	output := `Configuration for user@proton.me
IMAP Settings
Address:   127.0.0.1
IMAP port: 1143
Username:  user@proton.me
Password:  abcdef123456
Security:  STARTTLS

SMTP Settings
Address:   127.0.0.1
SMTP port: 1025
Username:  user@proton.me
Password:  abcdef123456
Security:  STARTTLS

>>> `

	accounts := parseAccounts(output)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	a := accounts[0]
	if a.Email != "user@proton.me" {
		t.Errorf("email = %q, want %q", a.Email, "user@proton.me")
	}
	if a.Username != "user@proton.me" {
		t.Errorf("username = %q, want %q", a.Username, "user@proton.me")
	}
	if a.Password != "abcdef123456" {
		t.Errorf("password = %q, want %q", a.Password, "abcdef123456")
	}
	if a.IMAPPort != 1143 {
		t.Errorf("imapPort = %d, want 1143", a.IMAPPort)
	}
	if a.SMTPPort != 1025 {
		t.Errorf("smtpPort = %d, want 1025", a.SMTPPort)
	}
}

func TestParseAccounts_Empty(t *testing.T) {
	accounts := parseAccounts(">>> ")
	if len(accounts) != 0 {
		t.Errorf("expected 0 accounts from empty output, got %d", len(accounts))
	}
}

func TestParseAccounts_NoiseOutput(t *testing.T) {
	// Bridge CLI often includes warnings and banner text
	output := `WARN Failed to check updates
INFO Bridge version 3.22.0

Account: user@pm.me
Username: user@pm.me
Password: bridgepass
IMAP port: 1143
SMTP port: 1025
Security: STARTTLS

>>> `

	accounts := parseAccounts(output)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	if accounts[0].Email != "user@pm.me" {
		t.Errorf("email = %q", accounts[0].Email)
	}
}

func TestParseListOutput(t *testing.T) {
	output := `#   account              status           address mode
0:  user@proton.me       connected        combined
>>> `

	entries := parseListOutput(output)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Index != "0" {
		t.Errorf("index = %q, want %q", entries[0].Index, "0")
	}
	if entries[0].Email != "user@proton.me" {
		t.Errorf("email = %q, want %q", entries[0].Email, "user@proton.me")
	}
	if entries[0].Status != "connected" {
		t.Errorf("status = %q, want %q", entries[0].Status, "connected")
	}
	if entries[0].Mode != "combined" {
		t.Errorf("mode = %q, want %q", entries[0].Mode, "combined")
	}
}

func TestParseListOutput_MultipleAccounts(t *testing.T) {
	output := `#   account              status           address mode
0:  alice@proton.me      connected        combined
1:  bob@pm.me            connected        split
>>> `

	entries := parseListOutput(output)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Email != "alice@proton.me" {
		t.Errorf("first email = %q", entries[0].Email)
	}
	if entries[1].Email != "bob@pm.me" {
		t.Errorf("second email = %q", entries[1].Email)
	}
	if entries[1].Mode != "split" {
		t.Errorf("second mode = %q", entries[1].Mode)
	}
}

func TestParseListOutput_Empty(t *testing.T) {
	entries := parseListOutput(">>> ")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseListOutput_ActualPTYFormat(t *testing.T) {
	// This is the actual format from the bridge CLI with PTY echo
	output := `>>> l >>> li >>> lis >>> list >>> list >>> list
 # : account              (status         , address mode   )
0 : pattertj             (locked         , combined       )

>>> `

	entries := parseListOutput(output)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Index != "0" {
		t.Errorf("index = %q, want %q", entries[0].Index, "0")
	}
	if entries[0].Email != "pattertj" {
		t.Errorf("email = %q, want %q", entries[0].Email, "pattertj")
	}
	if entries[0].Status != "locked" {
		t.Errorf("status = %q, want %q", entries[0].Status, "locked")
	}
	if entries[0].Mode != "combined" {
		t.Errorf("mode = %q, want %q", entries[0].Mode, "combined")
	}
}

func TestParseInfoBlock(t *testing.T) {
	output := `Configuration for user@proton.me
IMAP Settings
Address:   127.0.0.1
IMAP port: 1143
Username:  user@proton.me
Password:  abcdef123456
Security:  STARTTLS

SMTP Settings
Address:   127.0.0.1
SMTP port: 1025
Username:  user@proton.me
Password:  abcdef123456
Security:  STARTTLS

>>> `

	entry := listEntry{Index: "0", Email: "user@proton.me", Status: "connected", Mode: "combined"}
	acct := parseInfoBlock(output, entry)

	if acct.ID != "0" {
		t.Errorf("ID = %q, want %q", acct.ID, "0")
	}
	if acct.Email != "user@proton.me" {
		t.Errorf("email = %q, want %q", acct.Email, "user@proton.me")
	}
	if acct.Username != "user@proton.me" {
		t.Errorf("username = %q, want %q", acct.Username, "user@proton.me")
	}
	if acct.Password != "abcdef123456" {
		t.Errorf("password = %q, want %q", acct.Password, "abcdef123456")
	}
	if acct.IMAPPort != 1143 {
		t.Errorf("imapPort = %d, want 1143", acct.IMAPPort)
	}
	if acct.SMTPPort != 1025 {
		t.Errorf("smtpPort = %d, want 1025", acct.SMTPPort)
	}
	if acct.Security != "STARTTLS" {
		t.Errorf("security = %q, want STARTTLS", acct.Security)
	}
	if acct.Status != "connected" {
		t.Errorf("status = %q, want connected", acct.Status)
	}
	if acct.Mode != "combined" {
		t.Errorf("mode = %q, want combined", acct.Mode)
	}
}

func TestParseInfoBlock_FallbackEmail(t *testing.T) {
	// If list entry has no email, parseInfoBlock should extract from Configuration header
	output := `Configuration for fallback@proton.me
Username:  fallback@proton.me
Password:  secret
>>> `

	entry := listEntry{Index: "0", Email: "", Status: "connected", Mode: "combined"}
	acct := parseInfoBlock(output, entry)

	if acct.Email != "fallback@proton.me" {
		t.Errorf("email = %q, want %q", acct.Email, "fallback@proton.me")
	}
}

func TestSplitAccountBlocks(t *testing.T) {
	output := `Account: a@test.com
Password: p1

Account: b@test.com
Password: p2
`
	blocks := splitAccountBlocks(output)
	if len(blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(blocks))
	}
}

func TestSplitAccountBlocks_ConfigurationFor(t *testing.T) {
	output := `Configuration for user@proton.me
Username: user@proton.me
Password: secret
`
	blocks := splitAccountBlocks(output)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0] == "" {
		t.Error("block should not be empty")
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"\x1b[33mWARN\x1b[0m some text", "WARN some text"},
		{"no escapes here", "no escapes here"},
		{"\x1b[31mERRO\x1b[0m[Feb 28] msg", "ERRO[Feb 28] msg"},
		{"line1\r\nline2", "line1\nline2"},         // \r stripped
		{"echo\b\bback", "echoback"},               // \b stripped
		{"normal text", "normal text"},
	}

	for _, tt := range tests {
		got := stripANSI(tt.input)
		if got != tt.want {
			t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncate("hello world", 5); got != "hello..." {
		t.Errorf("truncate long = %q, want %q", got, "hello...")
	}
	if got := truncate("", 5); got != "" {
		t.Errorf("truncate empty = %q", got)
	}
}

func TestNewBridgeManagerDefaults(t *testing.T) {
	bm := NewBridgeManager()
	if bm.GetState() != StateRunning {
		t.Errorf("initial state = %v, want StateRunning", bm.GetState())
	}
	prog := bm.GetSyncProgress()
	if len(prog) != 0 {
		t.Errorf("initial sync progress should be empty, got %d", len(prog))
	}
}

func TestBridgeManagerClose(t *testing.T) {
	bm := NewBridgeManager()
	// Close should not panic even with no CLI
	bm.Close()
	// Should be safe to call twice
	bm.Close()
}

func TestSplitAccountBlocks_NoAccounts(t *testing.T) {
	blocks := splitAccountBlocks("some random output\nno accounts here\n>>> ")
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks from output with no @, got %d", len(blocks))
	}
}

func TestSplitAccountBlocks_SingleBlock(t *testing.T) {
	output := `Account: single@test.com
Username: single@test.com
Password: p1
`
	blocks := splitAccountBlocks(output)
	if len(blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocks))
	}
}

func TestParseAccounts_PartialData(t *testing.T) {
	// Account line but missing other fields
	output := `Account: partial@proton.me
>>> `
	accounts := parseAccounts(output)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	if accounts[0].Email != "partial@proton.me" {
		t.Errorf("email = %q", accounts[0].Email)
	}
	// Defaults should apply
	if accounts[0].Security != "STARTTLS" {
		t.Errorf("security default = %q", accounts[0].Security)
	}
	if accounts[0].IMAPPort != 1143 {
		t.Errorf("imapPort default = %d", accounts[0].IMAPPort)
	}
}

func TestParseSyncProgress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantN   int // number of matches
		account string
		pct     float64
	}{
		{
			name:    "standard sync line",
			input:   "Sync (pattertj): 5.2% (Elapsed: 0.4s, ETA: 6.8s)",
			wantN:   1,
			account: "pattertj",
			pct:     5.2,
		},
		{
			name:    "100% sync",
			input:   "Sync (user): 100.0% (Elapsed: 30.0s, ETA: 0.0s)",
			wantN:   1,
			account: "user",
			pct:     100.0,
		},
		{
			name:    "sync with email account",
			input:   "Sync (alice@proton.me): 42.7% (Elapsed: 5.0s, ETA: 7.0s)",
			wantN:   1,
			account: "alice@proton.me",
			pct:     42.7,
		},
		{
			name:  "no sync line",
			input: ">>> list\n0 : user (connected, combined)\n>>> ",
			wantN: 0,
		},
		{
			name:  "empty input",
			input: "",
			wantN: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := syncProgressRegex.FindAllStringSubmatch(tt.input, -1)
			if len(matches) != tt.wantN {
				t.Fatalf("expected %d matches, got %d", tt.wantN, len(matches))
			}
			if tt.wantN > 0 {
				if matches[0][1] != tt.account {
					t.Errorf("account = %q, want %q", matches[0][1], tt.account)
				}
				var pct float64
				fmt.Sscanf(matches[0][2], "%f", &pct)
				if pct != tt.pct {
					t.Errorf("pct = %f, want %f", pct, tt.pct)
				}
			}
		})
	}
}

func TestUpdateSyncProgress(t *testing.T) {
	bm := NewBridgeManager()

	bm.updateSyncProgress("testuser", 25.5)
	progress := bm.GetSyncProgress()

	if len(progress) != 1 {
		t.Fatalf("expected 1 progress entry, got %d", len(progress))
	}
	p := progress["testuser"]
	if p == nil {
		t.Fatal("expected progress for 'testuser'")
	}
	if p.Account != "testuser" {
		t.Errorf("account = %q, want %q", p.Account, "testuser")
	}
	if p.Percentage != 25.5 {
		t.Errorf("percentage = %f, want 25.5", p.Percentage)
	}
	if !p.Syncing {
		t.Error("expected syncing = true for 25.5%")
	}

	// Update to 100% — should mark as not syncing
	bm.updateSyncProgress("testuser", 100.0)
	progress = bm.GetSyncProgress()
	p = progress["testuser"]
	if p.Syncing {
		t.Error("expected syncing = false for 100%")
	}
}

func TestSyncBegunRegex(t *testing.T) {
	tests := []struct {
		input   string
		wantN   int
		account string
	}{
		{"A sync has begun for pattertj.", 1, "pattertj"},
		{"a sync has begun for user123.", 1, "user123"},
		{"no sync here", 0, ""},
	}
	for _, tt := range tests {
		m := syncBegunRegex.FindStringSubmatch(tt.input)
		if tt.wantN == 0 {
			if m != nil {
				t.Errorf("expected no match for %q, got %v", tt.input, m)
			}
		} else {
			if m == nil {
				t.Fatalf("expected match for %q", tt.input)
			}
			got := strings.TrimSuffix(m[1], ".")
			if got != tt.account {
				t.Errorf("account = %q, want %q", got, tt.account)
			}
		}
	}
}

func TestSyncFinishedRegex(t *testing.T) {
	tests := []struct {
		input   string
		wantN   int
		account string
	}{
		{"A sync has finished for pattertj.", 1, "pattertj"},
		{"a sync has finished for user123.", 1, "user123"},
		{"no sync here", 0, ""},
	}
	for _, tt := range tests {
		m := syncFinishedRegex.FindStringSubmatch(tt.input)
		if tt.wantN == 0 {
			if m != nil {
				t.Errorf("expected no match for %q, got %v", tt.input, m)
			}
		} else {
			if m == nil {
				t.Fatalf("expected match for %q", tt.input)
			}
			got := strings.TrimSuffix(m[1], ".")
			if got != tt.account {
				t.Errorf("account = %q, want %q", got, tt.account)
			}
		}
	}
}

func TestBridgeStateString(t *testing.T) {
	tests := []struct {
		state BridgeState
		want  string
	}{
		{StateRunning, "running"},
		{StateSetupMode, "setup"},
		{StateTransitioning, "transitioning"},
		{StateStopped, "stopped"},
		{BridgeState(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("BridgeState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
