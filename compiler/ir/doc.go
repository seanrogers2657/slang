// Package ir defines the Intermediate Representation for the Slang compiler.
//
// The IR uses Static Single Assignment (SSA) form where each value is assigned
// exactly once. This enables simple use-def chains and straightforward
// optimization passes.
//
// # Structure
//
// The IR is organized hierarchically:
//
//   - Program: Top-level container holding functions, structs, and globals
//   - Function: A function with parameters, return type, and basic blocks
//   - Block: A basic block containing a sequence of values and a terminator
//   - Value: An SSA value representing a single computation
//
// # SSA Form
//
// In SSA form, each Value is the result of exactly one operation. When control
// flow merges (e.g., after an if/else), Phi nodes are used to select between
// values from different predecessors:
//
//	b1:
//	    v1 = Const 1
//	    Jump -> b3
//
//	b2:
//	    v2 = Const 2
//	    Jump -> b3
//
//	b3:
//	    v3 = Phi [v1 from b1, v2 from b2]
//
// # Control Flow
//
// Basic blocks end with exactly one terminator:
//   - Jump: Unconditional jump to another block
//   - Branch: Conditional branch to one of two blocks
//   - Return: Return from function
//   - Exit: Exit the program
//
// # Type System
//
// The IR has its own type system that maps from Slang types:
//   - IntType: Signed/unsigned integers of various bit widths
//   - BoolType: Boolean values
//   - PtrType: Pointers to other types
//   - ArrayType: Fixed-size arrays
//   - StructType: User-defined structs with computed field offsets
//   - NullableType: Nullable wrapper types
//   - FuncType: Function signatures
//
// # Usage
//
//	// Generate IR from typed AST
//	gen := ir.NewGenerator()
//	prog, err := gen.Generate(typedAST)
//
//	// Print IR for debugging
//	ir.NewPrinter(os.Stdout).PrintProgram(prog)
//
//	// Validate IR well-formedness
//	if errs := ir.Validate(prog); len(errs) > 0 {
//	    // handle errors
//	}
//
//	// Generate assembly via backend
//	backend := arm64.NewBackend()
//	asm, err := backend.Generate(prog)
package ir
