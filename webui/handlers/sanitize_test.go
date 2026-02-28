package handlers

import "testing"

func TestSanitizeAccountID(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"0", false},
		{"123", false},
		{"abc-def", false},
		{"user1", false},
		{"ABC123", false},
		// Injection attempts
		{"0\ndelete 0", true},
		{"0; rm -rf /", true},
		{"0\rlogout 1", true},
		{"", true},
		{"hello world", true},
		{"0\tinfo", true},
		{"../../../etc/passwd", true},
		{"id$(whoami)", true},
	}

	for _, tt := range tests {
		_, err := sanitizeAccountID(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("sanitizeAccountID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}
