# Slang Testing Framework

This document describes the comprehensive testing framework for the Slang compiler project.

## Overview

The testing framework covers all major components of the Slang compiler:

- **Lexer** (frontend/lexer) - Tokenization of source code
- **Parser** (frontend/parser) - AST construction
- **Semantic Analyzer** (frontend/semantic) - Type checking and validation
- **Code Generator** (backend/codegen) - ARM64 assembly generation
- **Assembler** (assembler/slasm) - Native ARM64 assembler
- **End-to-End Tests** (test/) - E2E tests for sl and slasm
- **Integration Tests** - Pipeline integration tests

## Test Coverage

Current test coverage statistics:

- **Backend (Assembly Generator)**: 62.5% coverage
- **Frontend (Lexer)**: 96.7% coverage
- **Frontend (Parser)**: 60.4% coverage
- **Frontend (Semantic Analyzer)**: 62.2% coverage
- **Frontend (Error Framework)**: 89.6% coverage

## Running Tests

### Quick Test Commands

```bash
# Run all tests
go run cmd/slm/main.go test

# Run all tests with verbose output
go run cmd/slm/main.go test-verbose

# Generate coverage report
go run cmd/slm/main.go test-coverage

# Run specific component tests
go run cmd/slm/main.go test-integration
```

### Using Go Directly

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Run specific package tests
go test ./frontend/lexer/...
go test ./frontend/parser/...
go test ./frontend/semantic/...
go test ./backend/codegen/...
go test ./assembler/slasm/...
go test ./test/sl/...      # E2E tests for sl compiler
go test ./test/slasm/...   # E2E tests for slasm assembler

# Run tests with race detector
go test ./... -race

# Run specific test by name
go test -run TestLexerNumbers
go test -run TestEndToEndCompilation
go test ./test/sl/... -run TestE2E

# Run assembler encoder tests
go test ./assembler/slasm/... -run TestEncode
```

## Test Organization

### Lexer Tests (`frontend/lexer/lexer_test.go`)

Tests for the tokenization stage of the compiler:

- **TestLexerNumbers**: Number parsing (single/multiple digits, zero, large numbers)
- **TestLexerArithmeticOperators**: Arithmetic operators (+, -, *, /, %)
- **TestLexerComparisonOperators**: Comparison operators (==, !=, <, >, <=, >=)
- **TestLexerExpressions**: Complete expressions with multiple tokens
- **TestLexerErrors**: Error handling for invalid input
- **TestLexerWhitespace**: Whitespace handling (spaces, tabs, mixed)

### Parser Tests (`frontend/parser/parser_test.go`)

Tests for the AST construction stage:

- **TestParserLiterals**: Number literal parsing
- **TestParserBinaryExpressions**: Binary expressions with arithmetic operators
- **TestParserComparisonExpressions**: Binary expressions with comparison operators
- **TestParserParse**: Top-level parse function
- **TestParserErrors**: Error handling for unsupported operations
- **TestParserIntegrationWithLexer**: Integration between lexer and parser

### Semantic Analyzer Tests (`frontend/semantic/analyzer_test.go`)

Tests for type checking and semantic validation:

- **TestAnalyzerArithmetic**: Type checking for arithmetic operators (+, -, *, /, %)
- **TestAnalyzerComparison**: Type checking for comparison operators (==, !=, <, >, <=, >=)
- **TestAnalyzerTypeErrors**: Detection of type mismatches
- **TestAnalyzerMultiStatement**: Analysis of multi-statement programs
- **TestAnalyzerMainFunction**: Validation of main function presence

### Code Generator Tests (`backend/codegen/codegen_test.go`)

Tests for ARM64 assembly generation:

- **TestAsGeneratorInterface**: AsGenerator interface testing with function declarations
- **TestGenerateProgramMultipleStatements**: Multi-statement code generation
- **TestGenerateVarDecl**: Variable declaration code generation
- **TestGenerateAssignStmt**: Assignment statement code generation

### Integration Tests (`integration_test.go`)

End-to-end tests for the complete compilation pipeline:

- **TestEndToEndCompilation**: Full pipeline from source to assembly
- **TestCompilationPipelineStages**: Individual stage verification
- **TestExampleFile**: Verification of example files
- **TestRegressions**: Edge cases and bug prevention

### Assembler Tests (`assembler/slasm/`)

Unit tests for the native ARM64 assembler:

**Lexer Tests** (`lexer_test.go`):
- **TestLexerDirectives**: Parsing of `.global`, `.align`, data directives
- **TestLexerLabels**: Label recognition and parsing
- **TestLexerInstructions**: Instruction mnemonic recognition
- **TestLexerRegisters**: Register parsing (x0-x30, sp, lr, xzr)
- **TestLexerImmediates**: Immediate value parsing (#123, #0x1a)
- **TestLexerMemoryOperands**: Memory operands with writeback (`[sp, #-16]!`, `[sp], #16`)

**Parser Tests** (`parser_test.go`):
- **TestParserProgram**: Full program parsing
- **TestParserInstructions**: Instruction operand parsing
- **TestParserDirectives**: Directive argument parsing

**Symbol Table Tests** (`symbols_test.go`):
- **TestSymbolDefine**: Symbol definition
- **TestSymbolLookup**: Symbol lookup
- **TestSymbolDuplicate**: Duplicate symbol detection

**Layout Tests** (`layout_test.go`):
- **TestLayoutAddresses**: Address assignment
- **TestLayoutAlignment**: Alignment handling

**Encoder Tests** (`encoder_test.go`):
- **TestEncodeAdd**: ADD instruction encoding
- **TestEncodeSub**: SUB instruction encoding
- **TestEncodeMul**: MUL instruction encoding
- **TestEncodeSdiv**: SDIV instruction encoding
- **TestEncodeMsub**: MSUB instruction encoding
- **TestEncodeCmp**: CMP instruction encoding
- **TestEncodeCset**: CSET instruction encoding
- **TestEncodeLdp**: LDP instruction encoding (signed offset, pre-indexed, post-indexed)
- **TestEncodeStp**: STP instruction encoding (signed offset, pre-indexed, post-indexed)
- **TestEncodeBranch**: Branch instructions (B, BL, B.cond, BR)
- **TestEncodeLdr**: LDR instruction encoding
- **TestEncodeStr**: STR instruction encoding
- **TestEncodeData**: Data directive encoding (.byte, .word, .quad, .asciz)

### End-to-End Tests (`test/`)

E2E tests are located in the `test/` directory and read example files from `_examples/`. Each test file uses `@test:` directives in header comments to specify expectations.

**Slang Compiler E2E Tests** (`test/sl/e2e_test.go`):
Reads example files from `_examples/slang/*.sl` and runs them through the compiler pipeline.

**Assembler E2E Tests** (`test/slasm/e2e_test.go`):
Reads example files from `_examples/arm64/*.s` and assembles/executes them.

**Shared Test Utilities** (`test/testutil/`):
- `expectations.go` - Parses `@test:` directives from file headers
- `expectations_test.go` - Tests for the expectation parser

#### `@test:` Directive Format

Example files use `@test:` directives in header comments:

```slang
// @test: exit_code=42
fn main() {
    42
}
```

```asm
; @test: exit_code=5
; @test: skip=reason to skip
.global _start
_start:
    mov x0, #5
    mov x16, #1
    svc #0x80
```

**Supported directives:**
| Directive | Description | Default |
|-----------|-------------|---------|
| `exit_code=N` | Expected exit code | 0 |
| `stdout=text` | Expected stdout output | (empty) |
| `stderr=text` | Expected stderr output | (empty) |
| `skip=reason` | Skip the test with a reason | (not skipped) |
| `expect_error=true` | Test expects a compilation error | false |
| `error_stage=stage` | Which stage should produce the error (lexer, parser, semantic) | (any) |
| `error_contains=text` | Error message should contain this text | (any) |

#### Adding New E2E Tests

1. Create an example file in the appropriate `_examples/` directory:
   - Slang: `_examples/slang/your_test.sl`
   - ARM64: `_examples/arm64/your_test.s`

2. Add `@test:` directives at the top of the file:
   ```slang
   // @test: exit_code=0
   fn main() {
       // your test code
   }
   ```

3. Run the tests:
   ```bash
   go test ./test/sl/...    # Slang e2e tests
   go test ./test/slasm/... # Assembler e2e tests
   ```

Tests run in parallel using `t.Parallel()` for efficiency.

## Test Patterns

The tests follow Go best practices:

### Table-Driven Tests

Most tests use the table-driven pattern for maintainability:

```go
tests := []struct {
    name     string
    input    string
    expected []Token
}{
    {
        name:  "simple addition",
        input: "2 + 5",
        expected: []Token{
            {Type: TokenTypeInteger, Value: "2"},
            {Type: TokenTypePlus, Value: "+"},
            {Type: TokenTypeInteger, Value: "5"},
        },
    },
    // More test cases...
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test implementation
    })
}
```

### Subtests

Tests use `t.Run()` for better organization and parallel execution:

```go
t.Run("specific test case", func(t *testing.T) {
    // Test code
})
```

### Error Testing

Tests verify both success and error cases:

```go
if len(l.Errors) == 0 {
    t.Fatal("expected error, got none")
}

if err.Error() != expectedError {
    t.Errorf("expected error %q, got %q", expectedError, err.Error())
}
```

## Adding New Tests

### For Lexer

Add tests to `frontend/lexer/lexer_test.go`:

```go
func TestLexerNewFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []Token
    }{
        // Add test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            l := NewLexer([]byte(tt.input))
            l.Parse()
            // Verify results
        })
    }
}
```

### For Parser

Add tests to `frontend/parser/parser_test.go`:

```go
func TestParserNewFeature(t *testing.T) {
    tests := []struct {
        name     string
        tokens   []lexer.Token
        expected *Expr
    }{
        // Add test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := NewParser(tt.tokens)
            expr := p.Parse()
            // Verify results
        })
    }
}
```

### For Code Generator

Add tests to `backend/codegen/codegen_test.go`:

```go
func TestCodegenNewFeature(t *testing.T) {
    tests := []struct {
        name            string
        statements      []ast.Statement
        expectedContent []string
    }{
        // Add test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            program := &ast.Program{
                Declarations: []ast.Declaration{
                    &ast.FunctionDecl{
                        Name:       "main",
                        ReturnType: "void",
                        Body:       &ast.BlockStmt{Statements: tt.statements},
                    },
                },
            }
            output, err := GenerateProgram(program, nil)
            // Verify results
        })
    }
}
```

### For Integration

Add tests to `integration_test.go`:

```go
func TestNewIntegrationScenario(t *testing.T) {
    source := "your slang code"

    // Run through lexer
    l := lexer.NewLexer([]byte(source))
    l.Parse()

    // Run through parser
    p := parser.NewParser(l.Tokens)
    expr := p.Parse()

    // Run through code generator
    generator := as.NewAsGenerator(expr)
    output, err := generator.Generate()

    // Verify results
}
```

### For E2E Tests

Add example files to `_examples/` with `@test:` directives:

**Slang E2E Test** (`_examples/slang/new_feature.sl`):
```slang
// @test: exit_code=0
fn main(): void {
    val x = 42
    print(x)
}
```

**ARM64 E2E Test** (`_examples/arm64/new_feature.s`):
```asm
; @test: exit_code=42
.global _start
.align 4
_start:
    mov x0, #42
    mov x16, #1
    svc #0x80
```

The tests will automatically discover new files in these directories.

## Continuous Integration

The testing framework is designed to be CI-friendly:

```yaml
# Example GitHub Actions workflow
- name: Run tests
  run: go run cmd/slm/main.go test

- name: Generate coverage
  run: go run cmd/slm/main.go test-coverage

- name: Check coverage threshold
  run: |
    coverage=$(go test ./... -coverprofile=coverage.out | grep "coverage:" | awk '{print $5}' | tr -d '%')
    if (( $(echo "$coverage < 70" | bc -l) )); then
      echo "Coverage $coverage% is below 70%"
      exit 1
    fi
```

## Debugging Tests

### Verbose Output

```bash
# Run tests with verbose output
go test ./... -v

# Run specific test with verbose output
go test -v -run TestLexerNumbers
```

### Test-Specific Output

```bash
# Run only failing tests
go test ./... -failfast

# Run tests multiple times to detect flakiness
go test ./... -count=100
```

### Coverage Analysis

```bash
# Generate HTML coverage report
go run cmd/slm/main.go test-coverage
open coverage.html

# View coverage in terminal
go tool cover -func=coverage.out
```

## Known Limitations

1. **Build command not tested**: The `sl build` command has hardcoded paths making it difficult to test in isolation
2. **File I/O not mocked**: Integration tests could benefit from filesystem mocking

## Future Improvements

- [x] Add tests for `sl run` command
- [x] Add file-based e2e tests with expectation parsing
- [x] Add tests that verify exit codes from executed programs
- [ ] Fix `sl build` command hardcoded paths and add tests
- [ ] Add benchmark tests for performance tracking
- [ ] Add fuzzing tests for robustness
- [ ] Add property-based testing
- [ ] Mock filesystem operations in integration tests
- [ ] Add mutation testing to verify test quality

## Contributing

When contributing code:

1. Write tests for new features
2. Ensure all tests pass: `go run cmd/slm/main.go test`
3. Maintain coverage above 70%: `go run cmd/slm/main.go test-coverage`
4. Follow existing test patterns
5. Add documentation for complex test scenarios

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Test Coverage](https://blog.golang.org/cover)
