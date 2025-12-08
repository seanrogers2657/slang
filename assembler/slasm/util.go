package slasm

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseRegister extracts the register number from a register name.
// Returns an error for invalid register names.
func ParseRegister(name string) (int, error) {
	// Handle special registers
	switch name {
	case "sp", "xzr", "wzr":
		return 31, nil
	case "lr":
		return 30, nil
	}

	if len(name) < 2 {
		return 0, fmt.Errorf("invalid register name: %s", name)
	}

	// Check for x0-x30 or w0-w30
	prefix := name[0]
	if prefix != 'x' && prefix != 'w' {
		return 0, fmt.Errorf("invalid register name: %s (must start with 'x' or 'w')", name)
	}

	// Parse the number
	numStr := name[1:]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid register name: %s (cannot parse number)", name)
	}

	if num < 0 || num > 30 {
		return 0, fmt.Errorf("invalid register number: %d (must be 0-30)", num)
	}

	return num, nil
}

// ParseInt parses a string to an integer, supporting decimal and hexadecimal formats.
// Supports negative numbers and 0x prefix for hex.
// This is a convenience wrapper around ParseInt64 with range checking for int.
func ParseInt(s string) (int, error) {
	result, err := ParseInt64(s)
	if err != nil {
		return 0, err
	}

	// Check for overflow (int is platform-dependent, could be 32 or 64 bits)
	if result > int64(^uint(0)>>1) || result < -int64(^uint(0)>>1)-1 {
		return 0, fmt.Errorf("number out of range for int: %s", s)
	}

	return int(result), nil
}

// ParseUint parses a string to an unsigned integer.
func ParseUint(s string) (uint64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	s = strings.TrimSpace(s)

	// Handle hex numbers (0x prefix)
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return strconv.ParseUint(s[2:], 16, 64)
	}

	return strconv.ParseUint(s, 10, 64)
}

// EncodeLittleEndian converts a 32-bit value to little-endian bytes
func EncodeLittleEndian(value uint32) []byte {
	return []byte{
		byte(value & 0xFF),
		byte((value >> 8) & 0xFF),
		byte((value >> 16) & 0xFF),
		byte((value >> 24) & 0xFF),
	}
}

// ParseInt64 parses a string to an int64, supporting decimal and hexadecimal formats.
// Supports negative numbers and 0x prefix for hex.
func ParseInt64(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	s = strings.TrimSpace(s)
	negative := false

	// Handle negative sign
	if s[0] == '-' {
		negative = true
		s = s[1:]
		if s == "" {
			return 0, fmt.Errorf("invalid number: just '-'")
		}
	}

	var result int64
	var err error

	// Handle hex numbers (0x prefix)
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		result, err = strconv.ParseInt(s[2:], 16, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hex number: %s (%w)", s, err)
		}
	} else {
		// Decimal
		result, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid decimal number: %s (%w)", s, err)
		}
	}

	if negative {
		result = -result
	}

	return result, nil
}

// IsRegister checks if a string is a valid ARM64 register name
func IsRegister(name string) bool {
	_, err := ParseRegister(name)
	return err == nil
}
