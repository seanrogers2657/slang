// Package testutil provides shared utilities for e2e testing.
package testutil

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// TestExpectation holds the expected behavior parsed from a test file's header comments.
type TestExpectation struct {
	// FilePath is the absolute path to the test file
	FilePath string

	// Name is the base name of the file without extension
	Name string

	// ExitCode is the expected exit code (default 0)
	ExitCode int

	// Stdout is the expected stdout output (optional)
	Stdout string

	// Stderr is the expected stderr output (optional)
	Stderr string

	// StderrContains is a substring that should appear in stderr (optional)
	StderrContains string

	// Skip indicates this test should be skipped, with the reason
	Skip string

	// ExpectError indicates this test expects a compilation error
	ExpectError bool

	// ErrorStage indicates which stage should produce the error (lexer, parser, semantic, codegen)
	ErrorStage string

	// ErrorContains is a substring that should appear in the error message
	ErrorContains string
}

// ParseExpectations reads a file and extracts test expectations from @test: comments.
// Comments can be // or ; style (for assembly files).
func ParseExpectations(filePath string) (*TestExpectation, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	exp := &TestExpectation{
		FilePath: filePath,
		Name:     strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		ExitCode: 0, // default
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for comment prefixes
		var content string
		if strings.HasPrefix(line, "//") {
			content = strings.TrimPrefix(line, "//")
		} else if strings.HasPrefix(line, ";") {
			content = strings.TrimPrefix(line, ";")
		} else if line == "" {
			// Skip empty lines at the top
			continue
		} else {
			// Stop parsing when we hit non-comment content
			break
		}

		content = strings.TrimSpace(content)
		if !strings.HasPrefix(content, "@test:") {
			continue
		}

		// Parse the directive
		directive := strings.TrimPrefix(content, "@test:")
		directive = strings.TrimSpace(directive)

		if err := parseDirective(exp, directive); err != nil {
			return nil, fmt.Errorf("failed to parse directive %q: %w", directive, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	return exp, nil
}

// parseDirective parses a single @test: directive and updates the expectation.
func parseDirective(exp *TestExpectation, directive string) error {
	// Handle key=value format
	parts := strings.SplitN(directive, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid directive format, expected key=value")
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "exit_code":
		code, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid exit_code: %w", err)
		}
		exp.ExitCode = code

	case "stdout":
		// Unescape common escape sequences
		exp.Stdout = unescapeString(value)

	case "stderr":
		exp.Stderr = unescapeString(value)

	case "stderr_contains":
		exp.StderrContains = unescapeString(value)

	case "skip":
		exp.Skip = value

	case "expect_error":
		exp.ExpectError = value == "true"

	case "error_stage":
		exp.ErrorStage = value

	case "error_contains":
		exp.ErrorContains = unescapeString(value)

	default:
		return fmt.Errorf("unknown directive key: %s", key)
	}

	return nil
}

// unescapeString handles common escape sequences in test expectations.
func unescapeString(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

// DiscoverTestFiles finds all files matching the given pattern in a directory,
// recursively searching subdirectories.
func DiscoverTestFiles(dir, pattern string) ([]string, error) {
	var matches []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %q: %w", dir, err)
	}

	return matches, nil
}

// LoadTestCases discovers and parses all test files in a directory.
func LoadTestCases(dir, pattern string) ([]*TestExpectation, error) {
	files, err := DiscoverTestFiles(dir, pattern)
	if err != nil {
		return nil, err
	}

	var expectations []*TestExpectation
	for _, file := range files {
		exp, err := ParseExpectations(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}
		// Compute a relative name that includes subdirectory
		relPath, err := filepath.Rel(dir, file)
		if err == nil {
			// Convert path separators to underscores and remove extension
			name := strings.TrimSuffix(relPath, filepath.Ext(relPath))
			name = strings.ReplaceAll(name, string(filepath.Separator), "/")
			exp.Name = name
		}
		expectations = append(expectations, exp)
	}

	return expectations, nil
}
