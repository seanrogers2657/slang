# Slang Documentation

This directory contains all documentation for the Slang compiler and its components.

## Command-Line Tools

- **[CMD_TOOLS.md](CMD_TOOLS.md)** - Complete guide to all command-line tools:
  - Compiler (`sl`) - Main Slang compiler
  - Build tool (`slm`) - Cross-platform build system
  - Assembler (`slasm`) - Standalone ARM64 assembler
  - Integration test runners (`it`, `slasm-it`)
  - Usage examples and workflows

## SLASM Assembler Documentation

The slasm assembler is a custom ARM64 assembler that generates Mach-O executables directly.

### Main Documentation

- **[SLASM_README.md](SLASM_README.md)** - Complete documentation including:
  - Supported instructions
  - Usage examples
  - Mach-O structure details
  - Testing and debugging

- **[SLASM_STATUS.md](SLASM_STATUS.md)** - Current implementation status:
  - What's implemented
  - Test coverage
  - Known issues
  - File structure

- **[SLASM_DEBUG_GUIDE.md](SLASM_DEBUG_GUIDE.md)** - Debugging tools and techniques:
  - Debug build program
  - Debug tests
  - Understanding pipeline output
  - Common issues and solutions

- **[SLASM_FIXES.md](SLASM_FIXES.md)** - Implementation fixes and solutions:
  - Historical issues and fixes
  - Instruction encoding reference
  - Current investigation status

- **[MINIMAL_IMPLEMENTATION_PLAN.md](MINIMAL_IMPLEMENTATION_PLAN.md)** - Development roadmap:
  - Implementation phases
  - Success criteria
  - Current status
  - Next steps

### Reference Documentation

- **[reference/ARM64.md](reference/ARM64.md)** - ARM64 instruction encoding reference
- **[reference/MACH-O.md](reference/MACH-O.md)** - Mach-O file format reference

## Quick Links

### For Users
- [Getting Started](SLASM_README.md#quick-start)
- [Supported Instructions](SLASM_README.md#supported-instructions)
- [Usage Examples](SLASM_README.md#usage)

### For Developers
- [Current Status](SLASM_STATUS.md)
- [Implementation Plan](MINIMAL_IMPLEMENTATION_PLAN.md)
- [Debug Guide](SLASM_DEBUG_GUIDE.md)

### For Debugging
- [Debug Tools](SLASM_DEBUG_GUIDE.md#quick-start)
- [Known Issues](SLASM_README.md#known-issues)
- [Troubleshooting](SLASM_DEBUG_GUIDE.md#debugging-common-issues)

## Current Status Summary

✅ **Working:**
- Complete ARM64 instruction set for arithmetic and comparison operations
- Valid Mach-O executable generation with all modern load commands
- LC_DYLD_CHAINED_FIXUPS with correct format
- LC_SYMTAB and LC_DYSYMTAB with minimal symbol table
- Code signing validation passes
- Generated binaries execute correctly

⚠️ **Current Limitations:**
- No data section support yet
- No branch instructions with label resolution
- Limited to instructions needed by Slang compiler

See [SLASM_STATUS.md](SLASM_STATUS.md) for detailed status information.
