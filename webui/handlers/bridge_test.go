package handlers

import "testing"

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
