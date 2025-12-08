package slasm

import (
	"strings"
	"testing"
)

func TestParseRegister(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		// Valid X registers
		{"x0", "x0", 0, false},
		{"x15", "x15", 15, false},
		{"x30", "x30", 30, false},

		// Valid W registers
		{"w0", "w0", 0, false},
		{"w15", "w15", 15, false},
		{"w30", "w30", 30, false},

		// Special registers
		{"sp", "sp", 31, false},
		{"xzr", "xzr", 31, false},
		{"wzr", "wzr", 31, false},
		{"lr", "lr", 30, false},

		// Invalid registers
		{"empty", "", 0, true},
		{"single char", "x", 0, true},
		{"invalid prefix", "z0", 0, true},
		{"out of range", "x31", 0, true},
		{"negative", "x-1", 0, true},
		{"non-numeric", "xABC", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRegister(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRegister(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseRegister(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		// Decimal
		{"zero", "0", 0, false},
		{"positive", "42", 42, false},
		{"negative", "-42", -42, false},
		{"large", "1234567", 1234567, false},

		// Hexadecimal
		{"hex zero", "0x0", 0, false},
		{"hex lowercase", "0xff", 255, false},
		{"hex uppercase", "0xFF", 255, false},
		{"hex mixed", "0xDeAdBeEf", 0xDeAdBeEf, false},

		// Whitespace
		{"with spaces", "  42  ", 42, false},
		{"negative with spaces", "  -42  ", -42, false},

		// Invalid
		{"empty", "", 0, true},
		{"just minus", "-", 0, true},
		{"letters", "abc", 0, true},
		{"mixed", "12abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseInt(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseUint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{"zero", "0", 0, false},
		{"decimal", "42", 42, false},
		{"hex", "0xFF", 255, false},
		{"large", "0xFFFFFFFF", 0xFFFFFFFF, false},

		// Invalid
		{"empty", "", 0, true},
		{"negative", "-1", 0, true},
		{"invalid hex", "0xGG", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUint(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUint(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseUint(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEncodeLittleEndian(t *testing.T) {
	tests := []struct {
		name  string
		input uint32
		want  []byte
	}{
		{"zero", 0x00000000, []byte{0x00, 0x00, 0x00, 0x00}},
		{"one", 0x00000001, []byte{0x01, 0x00, 0x00, 0x00}},
		{"full", 0xDEADBEEF, []byte{0xEF, 0xBE, 0xAD, 0xDE}},
		{"pattern", 0x12345678, []byte{0x78, 0x56, 0x34, 0x12}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeLittleEndian(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("EncodeLittleEndian(0x%08x) length = %v, want %v", tt.input, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("EncodeLittleEndian(0x%08x)[%d] = 0x%02x, want 0x%02x", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsRegister(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid x0", "x0", true},
		{"valid x30", "x30", true},
		{"valid sp", "sp", true},
		{"valid lr", "lr", true},
		{"invalid x31", "x31", false},
		{"invalid z0", "z0", false},
		{"invalid empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRegister(tt.input)
			if got != tt.want {
				t.Errorf("IsRegister(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseSpaceSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
		errMsg  string
	}{
		// Valid sizes
		{"zero", "0", 0, false, ""},
		{"small", "100", 100, false, ""},
		{"medium", "4096", 4096, false, ""},
		{"max allowed", "1048576", MaxSpaceDirectiveSize, false, ""},
		{"with whitespace", "  100  ", 100, false, ""},
		{"hex", "0x100", 256, false, ""},

		// Invalid - negative
		{"negative", "-1", 0, true, "cannot be negative"},
		{"negative large", "-1000", 0, true, "cannot be negative"},

		// Invalid - too large
		{"over max", "1048577", 0, true, "exceeds maximum"},
		{"way over max", "999999999999", 0, true, "exceeds maximum"},

		// Invalid - not a number
		{"empty", "", 0, true, "invalid size"},
		{"letters", "abc", 0, true, "invalid"},
		{"mixed", "100abc", 0, true, "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSpaceSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseSpaceSize(%q) expected error containing %q, got nil", tt.input, tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ParseSpaceSize(%q) error = %q, want error containing %q", tt.input, err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseSpaceSize(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSpaceSize(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
