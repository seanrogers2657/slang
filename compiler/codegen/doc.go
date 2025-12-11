// Package codegen generates ARM64 assembly code for the Slang compiler.
//
// The package provides TypedCodeGenerator for generating assembly from a
// type-checked program (semantic.TypedProgram). It handles proper register
// selection for different types (integers vs floats, signed vs unsigned),
// runtime overflow checks, and symbol table generation for stack traces.
//
// # Generated Assembly
//
// Generated assembly targets ARM64 macOS and follows these conventions:
//   - Stack alignment: 16 bytes (ARM64 ABI requirement)
//   - Frame pointer: x29
//   - Link register: x30
//   - Result register: x2 (integer), d0 (float)
//   - Syscall number: x16
//
// # Usage
//
//	gen := codegen.NewTypedCodeGenerator(typedProgram, sourceLines)
//	assembly, err := gen.Generate()
package codegen
