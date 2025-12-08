// Package codegen generates ARM64 assembly code for the Slang compiler.
//
// The package provides two code generators for different stages of the
// compilation pipeline:
//
//   - AsGenerator: Generates assembly from an untyped AST (ast.Program).
//     Used for simple programs without type annotations.
//
//   - TypedCodeGenerator: Generates assembly from a type-checked program
//     (semantic.TypedProgram). Handles proper register selection for
//     different types (integers vs floats, signed vs unsigned).
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
// For typed programs (recommended):
//
//	gen := codegen.NewTypedCodeGenerator(typedProgram, sourceLines)
//	assembly, err := gen.Generate()
//
// For untyped programs (legacy):
//
//	gen := codegen.NewAsGenerator(program, sourceLines, typedProgram)
//	assembly, err := gen.Generate()
package codegen
