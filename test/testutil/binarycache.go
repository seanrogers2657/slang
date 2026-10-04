package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/seanrogers2657/slang/assembler"
	"github.com/seanrogers2657/slang/assembler/slasm"
)

// BuildCached assembles source into an executable kept in a persistent cache
// keyed by the source's hash, and returns its path. macOS scans every new
// executable file on its first launch (~200ms, effectively serialized), so
// reusing the same file for an unchanged program removes that cost from
// repeat test runs. Set SLANG_E2E_NOCACHE=1 to build into tmpDir instead.
func BuildCached(source, tmpDir string) (string, error) {
	sum := sha256.Sum256([]byte(source))
	key := hex.EncodeToString(sum[:])

	cacheDir := ""
	if os.Getenv("SLANG_E2E_NOCACHE") == "" {
		if base, err := os.UserCacheDir(); err == nil {
			cacheDir = filepath.Join(base, "slang-e2e")
			if err := os.MkdirAll(cacheDir, 0o755); err != nil {
				cacheDir = ""
			}
		}
	}
	if cacheDir == "" {
		outputPath := filepath.Join(tmpDir, "test_"+key[:16])
		return outputPath, assemble(source, outputPath)
	}

	outputPath := filepath.Join(cacheDir, key)
	if _, err := os.Stat(outputPath); err == nil {
		return outputPath, nil
	}

	// Build beside the final path and rename into place, so a parallel test
	// with the same program never runs a partially written file.
	tmpPath := fmt.Sprintf("%s.tmp%d", outputPath, time.Now().UnixNano())
	if err := assemble(source, tmpPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return "", fmt.Errorf("caching test binary: %w", err)
	}
	return outputPath, nil
}

func assemble(source, outputPath string) error {
	return slasm.New().Build(source, assembler.BuildOptions{OutputPath: outputPath})
}
