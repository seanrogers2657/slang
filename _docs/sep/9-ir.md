# Status

IMPLEMENTED, 2026-01-27

# Summary/Motivation

Introduce an Intermediate Representation (IR) layer between semantic analysis and code generation to enable multiple backends (ARM64, AMD64, WebAssembly, interpreter), optimization passes, and cleaner separation of concerns. The IR uses Static Single Assignment (SSA) form to enable powerful optimizations while maintaining a clean, well-defined structure.

# Goals/Non-Goals

- [goal] Define a mid-level SSA-based IR that captures all Slang semantics
- [goal] Create an IR generator that converts TypedAST to IR
- [goal] Refactor ARM64 backend to consume IR instead of TypedAST
- [goal] Provide IR printer for debugging and visualization
- [goal] Provide IR validator to verify well-formedness
- [goal] Design pass infrastructure for future optimizations
- [future] AMD64 backend implementation
- [future] Interpreter implementation
- [future] WebAssembly backend
- [future] Optimization passes (constant propagation, dead code elimination, CSE)
- [non-goal] Register allocation in IR (backends handle this)
- [non-goal] Automatic parallelization or vectorization
- [non-goal] JIT compilation

# Architecture

## Compilation Pipeline

```
                              FRONTEND                                          BACKEND
    ┌─────────────────────────────────────────────────────────┐    ┌─────────────────────────────────┐
    │                                                         │    │                                 │
    │  ┌─────────┐    ┌────────┐    ┌──────────┐    ┌──────┐  │    │  ┌─────────┐    ┌───────────┐  │
    │  │  Lexer  │───▶│ Parser │───▶│ Semantic │───▶│  IR  │──┼───▶│  │ Passes  │───▶│  Backend  │  │
    │  └─────────┘    └────────┘    └──────────┘    │ Gen  │  │    │  │ (opts)  │    └─────┬─────┘  │
    │                                               └──────┘  │    │  └─────────┘          │        │
    │                                                         │    │                       │        │
    │   source.sl      tokens         AST      TypedAST   IR  │    │      IR (opt)         ▼        │
    │                                                         │    │               ┌───────────────┐│
    └─────────────────────────────────────────────────────────┘    │               │    Output     ││
                                                                   │               └───────────────┘│
                                                                   └─────────────────────────────────┘
```

## IR Generation Flow

```
    TypedAST                           SSA IR
    ────────                           ──────

    TypedProgram                       Program
         │                                │
         ├── TypedFunction ──────────▶   Function
         │        │                          │
         │        ├── params ────────▶      []*Value (OpArg)
         │        │                          │
         │        └── body ──────────▶      []*Block
         │             │                         │
         │             ├── stmt ─────▶          Block
         │             │    │                     ├── Values []*Value
         │             │    ├── if ───▶          │    ├── v0 = Const 10
         │             │    │                    │    ├── v1 = Const 20
         │             │    ├── while ▶          │    └── v2 = Add v0, v1
         │             │    │                    │
         │             │    └── expr ─▶          └── Terminator
         │             │                              ├── Jump
         │             └── ...                        ├── Branch
         │                                            └── Return
         └── TypedStruct ────────────▶   StructType (with computed offsets)
```

## Multi-Backend Architecture

```
                                    ┌─────────────────────────────────────────┐
                                    │              IR Program                 │
                                    │  ┌─────────────────────────────────┐   │
                                    │  │  Function: main                 │   │
                                    │  │    b0: v0=Const 1, v1=Const 2   │   │
                                    │  │        v2=Add v0,v1             │   │
                                    │  │        Return v2                │   │
                                    │  └─────────────────────────────────┘   │
                                    └───────────────┬─────────────────────────┘
                                                    │
                         ┌──────────────────────────┼──────────────────────────┐
                         │                          │                          │
                         ▼                          ▼                          ▼
              ┌─────────────────────┐   ┌─────────────────────┐   ┌─────────────────────┐
              │   ARM64 Backend     │   │   AMD64 Backend     │   │    Interpreter      │
              │                     │   │                     │   │                     │
              │  arch.Emitter       │   │  arch.Emitter       │   │  VM with stack      │
              │  (arm64.Emitter)    │   │  (amd64.Emitter)    │   │  and heap           │
              └──────────┬──────────┘   └──────────┬──────────┘   └──────────┬──────────┘
                         │                          │                          │
                         ▼                          ▼                          ▼
              ┌─────────────────────┐   ┌─────────────────────┐   ┌─────────────────────┐
              │   ARM64 Assembly    │   │   AMD64 Assembly    │   │   Direct Execution  │
              │                     │   │                     │   │                     │
              │   add x2, x0, x1    │   │   add rax, rdi      │   │   result = 1 + 2    │
              │   ret               │   │   ret               │   │   return 3          │
              └─────────────────────┘   └─────────────────────┘   └─────────────────────┘
```

## Optimization Pipeline (Future Work)

```
                    ┌─────────────────────────────────────────────────────────┐
                    │             Optimization Passes (FUTURE)                │
                    │                                                         │
    IR Program ────▶│  ┌──────────┐   ┌──────────┐   ┌──────────┐            │────▶ Optimized IR
                    │  │ ConstProp│──▶│   DCE    │──▶│   CSE    │──▶  ...    │
                    │  └──────────┘   └──────────┘   └──────────┘            │
                    │                                                         │
                    │  Initial implementation: no optimization passes         │
                    │  Future: -O0 (none), -O1 (basic), -O2 (full)           │
                    └─────────────────────────────────────────────────────────┘

    Note: This SEP implements the pass infrastructure but no actual passes.
    Optimization passes will be added in a future SEP.
```

## SSA Construction (Phi Nodes)

```
    Source Code                          Control Flow Graph                    SSA IR
    ───────────                          ──────────────────                    ──────

    var x = 1                                  b0                            b0:
    if cond {                                   │                              v0 = Const 1
        x = 2                            ┌──────┴──────┐                       Branch cond → b1, b2
    } else {                             ▼             ▼
        x = 3                           b1            b2                     b1:
    }                                    │             │                       v1 = Const 2
    print(x)                             │   x = 2    │   x = 3                Jump → b3
                                         │             │
                                         └──────┬──────┘                     b2:
                                                ▼                              v2 = Const 3
                                               b3                              Jump → b3
                                           print(x)
                                                                             b3:
                                          which x?                             v3 = Phi [v1←b1, v2←b2]
                                             ───▶                              Call print(v3)
```

## Memory Layout

```
    Stack Frame (ARM64)                      Heap Object (Point struct)
    ───────────────────                      ─────────────────────────

    High addresses                           ┌─────────────────────────┐
         │                                   │  x: s64 (8 bytes)       │ offset 0
         ▼                                   ├─────────────────────────┤
    ┌─────────────────┐                      │  y: s64 (8 bytes)       │ offset 8
    │  Previous FP    │ ◀── x29 (frame ptr)  └─────────────────────────┘
    ├─────────────────┤                            Total: 16 bytes
    │  Return Addr    │ ◀── x30
    ├─────────────────┤
    │  Local var 1    │ [x29, #-8]           Nullable Primitive (s64?)
    ├─────────────────┤                      ─────────────────────────
    │  Local var 2    │ [x29, #-16]
    ├─────────────────┤                      ┌─────────────────────────┐
    │  Spill slot     │ [x29, #-24]          │  tag: u64 (8 bytes)     │ 0=null, 1=value
    ├─────────────────┤                      ├─────────────────────────┤
    │     ...         │                      │  value: s64 (8 bytes)   │
    └─────────────────┘                      └─────────────────────────┘
         ▲                                         Total: 16 bytes
         │
    Low addresses (SP)
```

# File Organization

## Directory Structure

```
slang/
├── cmd/
│   └── sl/
│       └── main.go                 # CLI entry point (updated for IR)
│
├── compiler/
│   ├── lexer/
│   │   └── lexer.go               # Tokenization (unchanged)
│   │
│   ├── parser/
│   │   └── parser.go              # AST construction (unchanged)
│   │
│   ├── ast/
│   │   └── ast.go                 # AST types (unchanged)
│   │
│   ├── semantic/
│   │   ├── analyzer.go            # Type checking (unchanged)
│   │   ├── typed_ast.go           # TypedAST types (unchanged)
│   │   └── types.go               # Type definitions (unchanged)
│   │
│   ├── ir/                        # NEW: IR package
│   │   ├── doc.go                 # Package documentation
│   │   ├── types.go               # IR type system
│   │   ├── op.go                  # Operation codes (OpAdd, OpSub, etc.)
│   │   ├── value.go               # Value struct (SSA values)
│   │   ├── block.go               # Basic block struct
│   │   ├── func.go                # Function struct
│   │   ├── program.go             # Program struct (top-level)
│   │   ├── generator.go           # TypedAST → IR conversion
│   │   ├── printer.go             # IR pretty-printer
│   │   ├── validate.go            # IR well-formedness checks
│   │   │
│   │   ├── passes/                # Optimization passes (FUTURE)
│   │   │   └── pass.go            # Pass interface & manager (infrastructure only)
│   │   │   # Future files:
│   │   │   # ├── constprop.go     # Constant propagation
│   │   │   # ├── dce.go           # Dead code elimination
│   │   │   # ├── cse.go           # Common subexpression elimination
│   │   │   # └── copyprop.go      # Copy propagation
│   │   │
│   │   └── backend/               # Code generation backends
│   │       ├── backend.go         # Backend interface
│   │       │
│   │       ├── arm64/             # ARM64 native backend
│   │       │   ├── backend.go     # IR → ARM64 assembly
│   │       │   ├── emit.go        # Instruction emission
│   │       │   ├── phi.go         # Phi elimination
│   │       │   └── regalloc.go    # Register allocation (simple)
│   │       │
│   │       ├── amd64/             # AMD64 native backend
│   │       │   ├── backend.go     # IR → AMD64 assembly
│   │       │   ├── emit.go        # Instruction emission
│   │       │   ├── phi.go         # Phi elimination
│   │       │   └── regalloc.go    # Register allocation
│   │       │
│   │       └── interp/            # Interpreter backend
│   │           ├── interpreter.go # IR interpreter
│   │           ├── value.go       # Runtime values
│   │           └── heap.go        # Heap management
│   │
│   ├── arch/                      # Architecture abstraction (existing)
│   │   ├── arch.go                # Emitter interface
│   │   ├── arm64/
│   │   │   └── emitter.go         # ARM64 instruction emitter
│   │   └── amd64/
│   │       └── emitter.go         # AMD64 instruction emitter
│   │
│   ├── codegen/                   # DEPRECATED: Old codegen (to be removed)
│   │   ├── typed_codegen.go       # Direct TypedAST → assembly
│   │   └── ...                    # Other codegen files
│   │
│   └── runtime/
│       └── errors.go              # Runtime error definitions
│
└── test/
    └── ir/                        # NEW: IR tests
        ├── generator_test.go      # IR generation tests
        ├── passes_test.go         # Optimization pass tests
        └── backend_test.go        # Backend tests
```

## File Responsibilities

### Core IR Files

| File | Responsibility | Key Types/Functions |
|------|----------------|---------------------|
| `types.go` | IR type system | `Type`, `IntType`, `PtrType`, `StructType` |
| `op.go` | Operation definitions | `Op`, `OpAdd`, `OpLoad`, `OpPhi`, etc. |
| `value.go` | SSA value representation | `Value`, `PhiArg` |
| `block.go` | Basic blocks | `Block`, `BlockKind` |
| `func.go` | Functions | `Function` |
| `program.go` | Top-level program | `Program`, `Global` |
| `generator.go` | AST to IR conversion | `Generator`, `Generate()` |
| `printer.go` | Debug output | `Printer`, `PrintFunction()` |
| `validate.go` | IR verification | `Validate()` |

### Pass Files (Infrastructure Only - Passes are Future Work)

| File | Pass | Description |
|------|------|-------------|
| `pass.go` | Infrastructure | `Pass` interface, `PassManager` |

Future passes (not part of this SEP):
- `constprop.go` - Constant Propagation
- `dce.go` - Dead Code Elimination
- `cse.go` - Common Subexpression Elimination
- `copyprop.go` - Copy Propagation

### Backend Files

| File | Backend | Description |
|------|---------|-------------|
| `backend/backend.go` | Interface | `Backend` interface, `Options` |
| `arm64/backend.go` | ARM64 | Main ARM64 code generator |
| `arm64/emit.go` | ARM64 | Value-to-instruction mapping |
| `arm64/phi.go` | ARM64 | Phi node elimination |
| `amd64/backend.go` | AMD64 | Main AMD64 code generator |
| `interp/interpreter.go` | Interpreter | Direct IR execution |

## Import Graph

```
                                cmd/sl/main.go
                                      │
                                      ▼
                    ┌─────────────────────────────────────┐
                    │           compiler/ir              │
                    │  ┌──────────┐    ┌──────────────┐  │
                    │  │ generator│    │   program    │  │
                    │  └────┬─────┘    └──────────────┘  │
                    │       │                 ▲          │
                    │       ▼                 │          │
                    │  ┌──────────┐    ┌──────┴───────┐  │
                    │  │  value   │◀───│    func      │  │
                    │  └──────────┘    └──────────────┘  │
                    │       ▲                 ▲          │
                    │       │                 │          │
                    │  ┌────┴─────┐    ┌──────┴───────┐  │
                    │  │  block   │    │    types     │  │
                    │  └──────────┘    └──────────────┘  │
                    │       ▲                 ▲          │
                    │       │                 │          │
                    │  ┌────┴─────────────────┴───────┐  │
                    │  │            op                │  │
                    │  └──────────────────────────────┘  │
                    └─────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
            ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
            │ ir/backend/ │   │ ir/passes/  │   │ ir/printer  │
            │   arm64     │   │  (future)   │   │             │
            │  (amd64)*   │   │             │   │             │
            │  (interp)*  │   │             │   │             │
            └──────┬──────┘   └─────────────┘   └─────────────┘
                   │
                   │          * = future work
                   ▼
                   │
                   ▼
            ┌─────────────┐
            │ arch/arm64  │
            │ arch/amd64  │
            └─────────────┘
```

## Migration Path

```
Phase 1: Add IR alongside existing codegen
──────────────────────────────────────────

    cmd/sl/main.go
          │
          ├──▶ compiler/semantic ──▶ TypedAST
          │                              │
          │         ┌────────────────────┴────────────────────┐
          │         │                                         │
          │         ▼                                         ▼
          │    compiler/ir/generator                  compiler/codegen
          │         │                                  (existing, still used)
          │         ▼                                         │
          │    compiler/ir/backend/arm64                      │
          │         │                                         │
          │         ▼                                         ▼
          └──▶ output.s ◀─────────────────────────────────────┘

          Flag: --use-ir (opt-in during development)


Phase 2: IR becomes default, old codegen deprecated
───────────────────────────────────────────────────

    cmd/sl/main.go
          │
          ├──▶ compiler/semantic ──▶ TypedAST
          │                              │
          │                              ▼
          │                    compiler/ir/generator
          │                              │
          │                              ▼
          │                    compiler/ir/passes (optional)
          │                              │
          │              ┌───────────────┼───────────────┐
          │              ▼               ▼               ▼
          │         arm64/backend   amd64/backend   interp/
          │              │               │               │
          └──▶     output.s         output.s        (execute)

          compiler/codegen/ marked deprecated


Phase 3: Remove old codegen
───────────────────────────

    compiler/codegen/ deleted
    All code paths use IR
```

# Design Decisions

## 1. SSA Form

The IR uses Static Single Assignment form where each value is assigned exactly once. This enables:
- Simple use-def chains (each use points to exactly one definition)
- Trivial dead code elimination (no uses = dead)
- Straightforward constant propagation
- Clean optimization pass implementation

Phi nodes handle control flow merges:
```
b1:
    v1 = Const 1
    Jump b3

b2:
    v2 = Const 2
    Jump b3

b3:
    v3 = Phi [v1 from b1, v2 from b2]  // v3 is 1 or 2
```

## 2. Abstraction Level

Mid-level IR that:
- Preserves type information for optimization
- Lowers structs to explicit field offsets
- Lowers arrays to pointer + length
- Keeps control flow as basic blocks with explicit jumps
- Does NOT include machine-specific details (registers, calling conventions)

## 3. Value Representation

All computations produce `*Value` objects:
- Constants: `OpConst` with embedded value
- Computations: Operation + input values
- Phi nodes: Merge values from different control flow paths

## 4. Block Structure

Functions consist of basic blocks:
- Each block has a label, list of instructions, and terminator
- Terminators: `Jump`, `Branch`, `Return`, `Exit`
- No fall-through; all control flow is explicit

## 5. Type System Mapping

| Slang Type | IR Type |
|------------|---------|
| `s8`-`s128` | `IntType{Bits, Signed: true}` |
| `u8`-`u128` | `IntType{Bits, Signed: false}` |
| `bool` | `BoolType{}` |
| `string` | `PtrType{ByteType}` + length |
| `[T; N]` | `ArrayType{Elem, Len}` |
| `T?` | `NullableType{Inner}` |
| `*T`, `&T`, `&&T` | `PtrType{Elem}` |
| structs | `StructType{Fields with offsets}` |

## 6. Ownership in IR

Ownership semantics are verified during semantic analysis. The IR just tracks:
- Allocation sites (`OpAlloc`)
- Deallocation points (`OpFree`, inserted by IR generator)
- Copy operations (`OpCopy` for `.copy()`)

The IR generator inserts `OpFree` at scope exits for owned pointers.

## 7. Nullable Representation

Following SEP-3 (Nullability):
- Primitive nullables (`s64?`): Tagged union (tag + value)
- Reference nullables (`Struct?`): Nullable pointer (0 = null)

IR operations:
- `OpIsNull`: Check if nullable is null
- `OpUnwrap`: Extract value (assumes not null)
- `OpWrap`: Create nullable from value
- `OpWrapNull`: Create null value

# APIs

## Core IR Types

```go
// compiler/ir/value.go

// Value represents an SSA value - the result of exactly one instruction
type Value struct {
    ID       int         // unique identifier (v0, v1, v2, ...)
    Op       Op          // operation that produces this value
    Type     Type        // result type
    Args     []*Value    // input values (SSA edges)
    Block    *Block      // containing block

    // For Phi nodes
    PhiArgs  []*PhiArg   // (value, predecessor block) pairs

    // For constants
    AuxInt   int64       // immediate integer value
    AuxFloat float64     // immediate float value
    AuxStr   string      // string constant or label

    // For field/index access
    AuxField int         // field index or byte offset

    // Metadata
    Uses     []*Value    // values that use this value (back-edges)
    Pos      Position    // source position for error messages
}

// PhiArg represents one incoming edge to a phi node
type PhiArg struct {
    Value *Value
    From  *Block
}
```

## Operations

```go
// compiler/ir/op.go

type Op int

const (
    // Constants
    OpConst Op = iota    // constant value (AuxInt, AuxFloat, or AuxStr)
    OpArg                // function argument (AuxField = arg index)

    // Arithmetic (Args: [left, right], result in Value)
    OpAdd
    OpSub
    OpMul
    OpDiv
    OpMod
    OpNeg                // unary: Args: [operand]

    // Comparison (produce BoolType)
    OpEq
    OpNe
    OpLt
    OpLe
    OpGt
    OpGe

    // Logical (BoolType operands and result)
    OpAnd
    OpOr
    OpNot                // unary

    // Memory
    OpAlloc              // allocate memory (AuxInt = size, Type = pointer type)
    OpFree               // deallocate memory (Args: [ptr])
    OpLoad               // load from pointer (Args: [ptr])
    OpStore              // store to pointer (Args: [ptr, value]), no result
    OpCopy               // deep copy (Args: [ptr])

    // Struct/Array access
    OpFieldPtr           // get field pointer (Args: [struct_ptr], AuxField = offset)
    OpIndexPtr           // get element pointer (Args: [array_ptr, index])
    OpArrayLen           // get array length (Args: [array])

    // Nullable operations
    OpIsNull             // check if nullable is null (Args: [nullable])
    OpUnwrap             // extract value from nullable (Args: [nullable])
    OpWrap               // create nullable with value (Args: [value])
    OpWrapNull           // create null nullable (Type = nullable type)

    // Control flow (terminators)
    OpPhi                // SSA phi node (uses PhiArgs)
    OpCall               // function call (Args: [arg0, arg1, ...], AuxStr = func name)
    OpJump               // unconditional jump (AuxStr = target label)
    OpBranch             // conditional branch (Args: [cond], block has Then/Else)
    OpReturn             // return from function (Args: [value] or empty)
    OpExit               // program exit (Args: [code])

    // Type conversions
    OpZeroExt            // zero extend (Args: [value], Type = target type)
    OpSignExt            // sign extend
    OpTrunc              // truncate
    OpIntToFloat         // integer to float
    OpFloatToInt         // float to integer
)
```

## Blocks and Functions

```go
// compiler/ir/block.go

type BlockKind int

const (
    BlockPlain  BlockKind = iota  // single successor (Jump)
    BlockIf                        // two successors (Branch)
    BlockReturn                    // no successors (Return)
    BlockExit                      // no successors (Exit)
)

type Block struct {
    ID       int           // unique identifier (b0, b1, b2, ...)
    Kind     BlockKind
    Func     *Function

    Values   []*Value      // instructions (non-terminators)
    Control  *Value        // for BlockIf: the condition value

    // Control flow graph
    Preds    []*Block      // predecessor blocks
    Succs    []*Block      // successor blocks (0, 1, or 2)
}

// compiler/ir/func.go

type Function struct {
    Name       string
    Params     []*Value      // OpArg values
    ReturnType Type
    Blocks     []*Block      // b0 is entry block

    // For ID generation
    nextValue  int
    nextBlock  int
}

// compiler/ir/program.go

type Program struct {
    Functions []*Function
    Structs   []*StructType
    Globals   []*Global
    Strings   []string       // string constant pool
}

type Global struct {
    Name  string
    Type  Type
    Init  *Value  // initial value (constant)
}
```

## IR Generator

```go
// compiler/ir/generator.go

type Generator struct {
    program    *Program
    fn         *Function
    block      *Block

    // Variable tracking for SSA construction
    varDefs    map[string]*Value           // current def of each variable
    sealedBlocks map[*Block]bool           // blocks with all preds known
    incompletePhis map[*Block]map[string]*Value

    // Loop tracking for break/continue
    loopStack  []*LoopInfo
}

type LoopInfo struct {
    CondBlock *Block  // for continue
    EndBlock  *Block  // for break
}

func NewGenerator() *Generator
func (g *Generator) Generate(typed *semantic.TypedProgram) (*Program, error)
```

## IR Printer

```go
// compiler/ir/printer.go

type Printer struct {
    w io.Writer
}

func NewPrinter(w io.Writer) *Printer
func (p *Printer) PrintProgram(prog *Program)
func (p *Printer) PrintFunction(fn *Function)
func (p *Printer) PrintBlock(b *Block)
func (p *Printer) PrintValue(v *Value)
```

## Backend Interface

```go
// compiler/ir/backend/backend.go

type Backend interface {
    // Generate produces target code from IR
    Generate(prog *Program) ([]byte, error)

    // Target returns the compilation target
    Target() arch.Target
}

// compiler/ir/backend/arm64/backend.go
type ARM64Backend struct {
    emitter *arm64.Emitter
}

// compiler/ir/backend/amd64/backend.go
type AMD64Backend struct {
    emitter *amd64.Emitter
}

// compiler/ir/backend/interp/interpreter.go
type Interpreter struct {
    stack  []Value
    frames []*CallFrame
    heap   *Heap
}
```

# Description

## Phase 1: Core IR Infrastructure

### Step 1.1: Define IR Types

Create the core IR type definitions in `compiler/ir/`:

```
compiler/ir/
├── types.go        # IR type system (IntType, PtrType, etc.)
├── op.go           # Operation codes
├── value.go        # Value struct and methods
├── block.go        # Block struct and methods
├── func.go         # Function struct
├── program.go      # Program struct (top-level)
└── doc.go          # Package documentation
```

### Step 1.2: Implement IR Printer

Essential for debugging. Output format:

```
function add(v0: i64, v1: i64) -> i64 {
b0:                             ; entry
    v2 = Add v0, v1 [i64]
    Return v2
}

function main() -> void {
b0:                             ; entry
    v0 = Const 10 [i64]
    v1 = Const 20 [i64]
    v2 = Call add(v0, v1) [i64]
    v3 = Const 25 [i64]
    v4 = Gt v2, v3 [bool]
    Branch v4 -> b1, b2

b1:                             ; preds: b0
    v5 = Call print(v2) [void]
    Jump -> b2

b2:                             ; preds: b0, b1
    Return
}
```

### Step 1.3: Implement IR Validator

Verify IR well-formedness:
- Each value has exactly one definition
- All uses refer to values that dominate them
- Phi nodes have one arg per predecessor
- Blocks end with exactly one terminator
- Type consistency (operands match operation requirements)

```go
// compiler/ir/validate.go

func Validate(prog *Program) []error {
    var errs []error
    for _, fn := range prog.Functions {
        errs = append(errs, validateFunction(fn)...)
    }
    return errs
}
```

## Phase 2: IR Generator

### Step 2.1: Basic Expression Lowering

Convert TypedAST expressions to IR values:

```go
func (g *Generator) lowerExpr(expr semantic.TypedExpression) *Value {
    switch e := expr.(type) {
    case *semantic.TypedLiteralExpr:
        return g.lowerLiteral(e)
    case *semantic.TypedBinaryExpr:
        return g.lowerBinary(e)
    case *semantic.TypedIdentifierExpr:
        return g.readVariable(e.Name, e.Type)
    case *semantic.TypedCallExpr:
        return g.lowerCall(e)
    // ... etc
    }
}

func (g *Generator) lowerBinary(e *semantic.TypedBinaryExpr) *Value {
    left := g.lowerExpr(e.Left)
    right := g.lowerExpr(e.Right)

    op := binaryOpToIR(e.Op)  // "+" -> OpAdd, etc.
    return g.newValue(op, e.Type, left, right)
}
```

### Step 2.2: SSA Variable Handling

Implement the SSA construction algorithm (Braun et al.):

```go
// writeVariable records that 'name' now has value 'val'
func (g *Generator) writeVariable(name string, val *Value) {
    g.varDefs[name] = val
}

// readVariable gets current value, inserting phi if needed
func (g *Generator) readVariable(name string, typ Type) *Value {
    if val, ok := g.varDefs[name]; ok {
        return val
    }
    return g.readVariableRecursive(name, typ, g.block)
}

func (g *Generator) readVariableRecursive(name string, typ Type, block *Block) *Value {
    var val *Value

    if !g.sealedBlocks[block] {
        // Block not sealed - create incomplete phi
        phi := g.newPhi(typ, block)
        g.incompletePhis[block][name] = phi
        val = phi
    } else if len(block.Preds) == 0 {
        panic("undefined variable: " + name)  // Should be caught by semantic analysis
    } else if len(block.Preds) == 1 {
        // Single predecessor - no phi needed
        val = g.readVariableRecursive(name, typ, block.Preds[0])
    } else {
        // Multiple predecessors - need phi
        phi := g.newPhi(typ, block)
        g.writeVariable(name, phi)  // break cycles
        val = g.addPhiOperands(name, phi)
    }

    g.writeVariable(name, val)
    return val
}
```

### Step 2.3: Control Flow Lowering

Convert structured control flow to basic blocks:

**If statement:**
```go
func (g *Generator) lowerIf(stmt *semantic.TypedIfStmt) {
    cond := g.lowerExpr(stmt.Condition)

    thenBlock := g.newBlock()
    elseBlock := g.newBlock()
    endBlock := g.newBlock()

    // Branch
    g.block.Kind = BlockIf
    g.block.Control = cond
    g.block.Succs = []*Block{thenBlock, elseBlock}
    thenBlock.Preds = []*Block{g.block}
    elseBlock.Preds = []*Block{g.block}

    // Then branch
    g.block = thenBlock
    g.lowerBlock(stmt.ThenBranch)
    g.emitJump(endBlock)

    // Else branch
    g.block = elseBlock
    if stmt.ElseBranch != nil {
        g.lowerBlock(stmt.ElseBranch)
    }
    g.emitJump(endBlock)

    // Merge point
    g.block = endBlock
    g.sealBlock(endBlock)
}
```

**While loop:**
```go
func (g *Generator) lowerWhile(stmt *semantic.TypedWhileStmt) {
    condBlock := g.newBlock()
    bodyBlock := g.newBlock()
    endBlock := g.newBlock()

    g.pushLoop(condBlock, endBlock)
    defer g.popLoop()

    // Jump to condition
    g.emitJump(condBlock)

    // Condition block
    g.block = condBlock
    cond := g.lowerExpr(stmt.Condition)
    g.block.Kind = BlockIf
    g.block.Control = cond
    g.block.Succs = []*Block{bodyBlock, endBlock}

    // Body block
    g.block = bodyBlock
    g.sealBlock(bodyBlock)
    g.lowerBlock(stmt.Body)
    g.emitJump(condBlock)

    // Seal condition block after body (has back-edge)
    g.sealBlock(condBlock)

    // End block
    g.block = endBlock
    g.sealBlock(endBlock)
}
```

### Step 2.4: Function and Struct Handling

```go
func (g *Generator) lowerFunction(fn *semantic.TypedFunctionDecl) *Function {
    irFn := &Function{
        Name:       fn.Name,
        ReturnType: convertType(fn.ReturnType),
    }

    // Create entry block
    entry := irFn.newBlock()
    g.block = entry

    // Create OpArg values for parameters
    for i, param := range fn.Params {
        argVal := g.newValue(OpArg, convertType(param.Type))
        argVal.AuxField = i
        irFn.Params = append(irFn.Params, argVal)
        g.writeVariable(param.Name, argVal)
    }

    // Lower function body
    g.lowerBlock(fn.Body)

    // Ensure function ends with return
    if g.block.Kind == BlockPlain {
        g.newValue(OpReturn, nil)
        g.block.Kind = BlockReturn
    }

    return irFn
}
```

### Step 2.5: Memory Operations

```go
// Heap.new(Point{1, 2})
func (g *Generator) lowerHeapNew(call *semantic.TypedCallExpr) *Value {
    structType := call.Args[0].GetType().(*semantic.StructType)
    size := g.structSize(structType)

    // Allocate
    ptr := g.newValue(OpAlloc, &PtrType{Elem: convertType(structType)})
    ptr.AuxInt = int64(size)

    // Initialize fields
    structLit := call.Args[0].(*semantic.TypedStructLiteral)
    for i, field := range structLit.Fields {
        fieldPtr := g.newValue(OpFieldPtr, &PtrType{Elem: convertType(field.Type)}, ptr)
        fieldPtr.AuxField = g.fieldOffset(structType, i)

        val := g.lowerExpr(field.Value)
        g.newValue(OpStore, nil, fieldPtr, val)
    }

    return ptr
}

// Insert OpFree when owned pointer goes out of scope
func (g *Generator) lowerScopeExit(scope *Scope) {
    for _, varInfo := range scope.OwnedPointers {
        ptr := g.readVariable(varInfo.Name, varInfo.Type)
        g.newValue(OpFree, nil, ptr)
    }
}
```

## Phase 3: Backend Refactoring

### Step 3.1: Define Backend Interface

```go
// compiler/ir/backend/backend.go

type Backend interface {
    Generate(prog *ir.Program) ([]byte, error)
    Target() arch.Target
}

type Options struct {
    OptLevel    int     // 0 = none, 1 = basic, 2 = full
    Debug       bool    // include debug info
    PIC         bool    // position-independent code
}
```

### Step 3.2: ARM64 Backend from IR

Refactor existing codegen to consume IR instead of TypedAST:

```go
// compiler/ir/backend/arm64/backend.go

type Backend struct {
    emitter  *arm64.Emitter
    builder  strings.Builder
    valueReg map[*ir.Value]string  // which register holds this value
}

func (b *Backend) Generate(prog *ir.Program) ([]byte, error) {
    // Emit data section
    b.builder.WriteString(b.emitter.EmitDataSection(prog.UsesPrint()))

    // Emit runtime functions
    b.emitRuntime()

    // Emit each function
    for _, fn := range prog.Functions {
        b.generateFunction(fn)
    }

    return []byte(b.builder.String()), nil
}

func (b *Backend) generateFunction(fn *ir.Function) {
    b.builder.WriteString(b.emitter.EmitFunctionLabel(fn.Name))

    // Calculate stack size
    stackSize := b.calculateStackSize(fn)
    b.builder.WriteString(b.emitter.EmitFunctionPrologue(stackSize))

    // Generate each block
    for _, block := range fn.Blocks {
        b.generateBlock(block)
    }
}

func (b *Backend) generateBlock(block *ir.Block) {
    // Emit label
    b.builder.WriteString(fmt.Sprintf("_%s_b%d:\n", block.Func.Name, block.ID))

    // Generate phi nodes first (as parallel copies from preds)
    b.generatePhis(block)

    // Generate each value
    for _, v := range block.Values {
        b.generateValue(v)
    }

    // Generate terminator
    b.generateTerminator(block)
}

func (b *Backend) generateValue(v *ir.Value) {
    switch v.Op {
    case ir.OpConst:
        b.builder.WriteString(b.emitter.EmitMoveImm("x2", fmt.Sprint(v.AuxInt)))

    case ir.OpAdd:
        b.loadOperands(v.Args[0], v.Args[1])
        code, _ := b.emitter.EmitIntOp("+", v.Type.(*ir.IntType).Signed)
        b.builder.WriteString(code)

    case ir.OpLoad:
        // Load from pointer in x2
        b.generateValue(v.Args[0])
        b.builder.WriteString("    ldr x2, [x2]\n")

    case ir.OpStore:
        // Store value to pointer
        b.generateValue(v.Args[1])  // value to x2
        b.builder.WriteString("    str x2, [sp, #-16]!\n")  // save value
        b.generateValue(v.Args[0])  // pointer to x2
        b.builder.WriteString("    ldr x3, [sp], #16\n")     // restore value to x3
        b.builder.WriteString("    str x3, [x2]\n")          // store

    case ir.OpCall:
        b.generateCall(v)

    // ... etc
    }
}
```

### Step 3.3: Phi Elimination

Convert phi nodes to copies in predecessor blocks:

```go
func (b *Backend) generatePhis(block *ir.Block) {
    // Collect all phi nodes
    var phis []*ir.Value
    for _, v := range block.Values {
        if v.Op == ir.OpPhi {
            phis = append(phis, v)
        }
    }

    if len(phis) == 0 {
        return
    }

    // For each predecessor, insert copies at the end
    for i, pred := range block.Preds {
        for _, phi := range phis {
            // phi.PhiArgs[i].Value is the value coming from pred
            // We need to copy it to a temp location
            // This is handled during terminator generation
        }
    }
}
```

## Phase 4: Pass Infrastructure (No Actual Passes)

This SEP implements only the pass infrastructure. Actual optimization passes will be added in future SEPs.

### Step 4.1: Pass Interface and Manager

```go
// compiler/ir/passes/pass.go

// Pass defines the interface for optimization passes.
// Actual passes (constprop, dce, etc.) will be added in future SEPs.
type Pass interface {
    Name() string
    Run(fn *ir.Function) bool  // returns true if IR was modified
}

type PassManager struct {
    passes []Pass
}

func NewPassManager() *PassManager {
    return &PassManager{}
}

func (pm *PassManager) Add(p Pass) {
    pm.passes = append(pm.passes, p)
}

// Run executes all passes until no changes are made.
// Initially this is a no-op since no passes are registered.
func (pm *PassManager) Run(prog *ir.Program) {
    for _, fn := range prog.Functions {
        changed := true
        for changed {
            changed = false
            for _, pass := range pm.passes {
                if pass.Run(fn) {
                    changed = true
                }
            }
        }
    }
}
```

**Note:** The pass manager is functional but no passes are registered. This allows the pipeline to be complete while deferring optimization work.

---

## Future Work: Optimization Passes (Separate SEP)

The following passes are planned for future implementation:

- **Constant Propagation**: Fold `1 + 2` → `3` at compile time
- **Dead Code Elimination**: Remove values with no uses
- **Common Subexpression Elimination**: Reuse `a + b` if computed twice
- **Copy Propagation**: Eliminate `x = y` when x and y are equivalent

---

## Future Work: Additional Backends (Separate SEPs)

### AMD64 Backend

The AMD64 backend will follow the same structure as ARM64, using the existing `arch/amd64` emitter interface:

```go
// Future: compiler/ir/backend/amd64/backend.go
type Backend struct {
    emitter *amd64.Emitter
    builder strings.Builder
}
```

### Interpreter Backend

The interpreter will execute IR directly, enabling REPL and fast iteration:

```go
// Future: compiler/ir/backend/interp/interpreter.go
type Interpreter struct {
    prog   *ir.Program
    frames []*Frame
    heap   map[int64][]byte
}
```

### WebAssembly Backend

A WASM backend would enable browser execution and portable binaries.

# Implementation Order

## This SEP (Core IR Infrastructure)

| Phase | Task | Files | Status |
|-------|------|-------|--------|
| 1.1 | Core IR types | `compiler/ir/types.go`, `op.go`, `value.go`, `block.go`, `func.go`, `program.go` | |
| 1.2 | IR printer | `compiler/ir/printer.go` | |
| 1.3 | IR validator | `compiler/ir/validate.go` | |
| 2.1 | Expression lowering | `compiler/ir/generator.go` | |
| 2.2 | SSA variable handling | `compiler/ir/generator.go` | |
| 2.3 | Control flow lowering | `compiler/ir/generator.go` | |
| 2.4 | Function/struct handling | `compiler/ir/generator.go` | |
| 2.5 | Memory operations | `compiler/ir/generator.go` | |
| 3.1 | Backend interface | `compiler/ir/backend/backend.go` | |
| 3.2 | ARM64 backend from IR | `compiler/ir/backend/arm64/` | |
| 3.3 | Phi elimination | `compiler/ir/backend/arm64/phi.go` | |
| 3.4 | Pass infrastructure (no passes) | `compiler/ir/passes/pass.go` | |

## Future Work (Separate SEPs)

| Task | Description | Depends On |
|------|-------------|------------|
| Constant propagation | Fold constant expressions at compile time | This SEP |
| Dead code elimination | Remove unused values and blocks | This SEP |
| Common subexpression elimination | Reuse identical computations | This SEP |
| Copy propagation | Eliminate redundant copies | This SEP |
| AMD64 backend | x86-64 code generation | This SEP |
| Interpreter | Direct IR execution for REPL | This SEP |
| WebAssembly backend | Browser/portable execution | This SEP |

# Files Created/Modified

## New Files (This SEP)

```
compiler/ir/
├── doc.go              # Package documentation
├── types.go            # IR type system
├── op.go               # Operation codes
├── value.go            # Value struct
├── block.go            # Block struct
├── func.go             # Function struct
├── program.go          # Program struct
├── generator.go        # TypedAST -> IR
├── printer.go          # IR pretty-printer
├── validate.go         # IR validation
├── passes/
│   └── pass.go         # Pass interface & manager (infrastructure only)
└── backend/
    ├── backend.go      # Backend interface
    └── arm64/
        ├── backend.go  # ARM64 backend (IR -> assembly)
        ├── emit.go     # Instruction emission helpers
        └── phi.go      # Phi node elimination
```

## Future Files (Not This SEP)

```
compiler/ir/
├── passes/
│   ├── constprop.go    # Future: Constant propagation
│   ├── dce.go          # Future: Dead code elimination
│   └── cse.go          # Future: Common subexpression elimination
└── backend/
    ├── amd64/
    │   └── backend.go  # Future: AMD64 backend
    └── interp/
        └── interpreter.go  # Future: IR interpreter
```

## Modified Files

| File | Changes |
|------|---------|
| `cmd/sl/main.go` | Add IR generation step, backend selection, `--use-ir` flag |
| `compiler/codegen/typed_codegen.go` | Mark as deprecated (still functional during transition) |

# Alternatives Considered

## 1. LLVM IR

**Pros:**
- Industry standard, extremely mature
- Many targets for free
- World-class optimizations

**Cons:**
- Large dependency (~100MB)
- Complex API, steep learning curve
- Opaque - harder to debug and understand
- Less educational value

**Decision:** Build custom IR for learning and control. LLVM could be added as an optional backend later.

## 2. Stack-based IR (like JVM bytecode)

**Pros:**
- Simpler, no register allocation
- Easy interpreter implementation

**Cons:**
- Less amenable to optimizations
- Harder to map to register machines efficiently

**Decision:** Register-based SSA is more powerful and maps better to real hardware.

## 3. No SSA (simple three-address code)

**Pros:**
- Simpler implementation
- No phi nodes to handle

**Cons:**
- Harder to implement optimizations
- Complex use-def chains
- More difficult data flow analysis

**Decision:** SSA is worth the complexity for the optimization benefits.

## 4. Continue extending current architecture

**Pros:**
- No new abstraction layer
- Works today

**Cons:**
- Code duplication for each backend
- Optimization must be duplicated per-backend
- Already showing strain with just ARM64

**Decision:** IR provides cleaner separation and enables reuse.

# Risks and Limitations

1. **Implementation Complexity**: SSA construction algorithm is non-trivial. The Braun et al. algorithm helps but still requires careful implementation.

2. **Performance Overhead**: IR adds a compilation phase. For small programs, may be noticeable. Can be mitigated with lazy/incremental generation.

3. **Debugging Difficulty**: Another abstraction layer means more places for bugs to hide. IR printer and validator are essential.

4. **Phi Node Lowering**: Converting phi nodes to actual code (parallel copies) requires careful handling to avoid correctness issues.

5. **Memory Operations**: Correctly sequencing loads/stores around phi elimination and optimization is subtle.

6. **Testing Burden**: Each component needs thorough testing. IR generator, backends, passes all need dedicated test suites.

# Testing

## Unit Tests

- **IR construction**: Build IR programmatically, verify structure
- **IR printer**: Round-trip test (print → parse → print)
- **Validator**: Test detection of malformed IR
- **Each optimization pass**: Test on hand-crafted IR inputs

## Integration Tests

- **IR generator**: Compare TypedAST programs against expected IR output
- **Full pipeline**: Source → IR → backend → execute → verify output

## E2E Tests

All existing E2E tests in `_examples/slang/` should continue to pass when routed through IR.

New tests for IR-specific behavior:
- `_examples/slang/ir/` - Programs that exercise IR edge cases
- Multiple backend comparison (ARM64 vs interpreted should give same results)

## Fuzzing

- Generate random valid IR programs
- Verify validator doesn't crash
- Verify backends produce runnable code

# Code Examples

## Example 1: Simple Function

**Source:**
```slang
add = (a: s64, b: s64) -> s64 {
    return a + b
}
```

**IR:**
```
function add(v0: i64, v1: i64) -> i64 {
b0:
    v2 = Add v0, v1 [i64]
    Return v2
}
```

## Example 2: If Statement

**Source:**
```slang
max = (a: s64, b: s64) -> s64 {
    if a > b {
        return a
    } else {
        return b
    }
}
```

**IR:**
```
function max(v0: i64, v1: i64) -> i64 {
b0:
    v2 = Gt v0, v1 [bool]
    Branch v2 -> b1, b2

b1:                             ; preds: b0
    Return v0

b2:                             ; preds: b0
    Return v1
}
```

## Example 3: While Loop with Phi

**Source:**
```slang
sum = (n: s64) -> s64 {
    var i = 0
    var total = 0
    while i < n {
        total = total + i
        i = i + 1
    }
    return total
}
```

**IR:**
```
function sum(v0: i64) -> i64 {
b0:                             ; entry
    v1 = Const 0 [i64]          ; initial i
    v2 = Const 0 [i64]          ; initial total
    Jump -> b1

b1:                             ; loop header, preds: b0, b2
    v3 = Phi [v1 from b0, v7 from b2] [i64]   ; i
    v4 = Phi [v2 from b0, v6 from b2] [i64]   ; total
    v5 = Lt v3, v0 [bool]
    Branch v5 -> b2, b3

b2:                             ; loop body, preds: b1
    v6 = Add v4, v3 [i64]       ; total + i
    v8 = Const 1 [i64]
    v7 = Add v3, v8 [i64]       ; i + 1
    Jump -> b1

b3:                             ; exit, preds: b1
    Return v4
}
```

## Example 4: Struct and Heap Allocation

**Source:**
```slang
Point = struct {
    var x: s64
    var y: s64
}

main = () {
    val p = Heap.new(Point{10, 20})
    p.x = 100
    print(p.x)
}
```

**IR:**
```
function main() -> void {
b0:
    v0 = Const 16 [i64]         ; sizeof(Point)
    v1 = Alloc v0 [*Point]      ; allocate

    ; Initialize x
    v2 = FieldPtr v1, 0 [*i64]  ; &p.x (offset 0)
    v3 = Const 10 [i64]
    Store v2, v3

    ; Initialize y
    v4 = FieldPtr v1, 8 [*i64]  ; &p.y (offset 8)
    v5 = Const 20 [i64]
    Store v4, v5

    ; p.x = 100
    v6 = FieldPtr v1, 0 [*i64]
    v7 = Const 100 [i64]
    Store v6, v7

    ; print(p.x)
    v8 = FieldPtr v1, 0 [*i64]
    v9 = Load v8 [i64]
    v10 = Call print(v9) [void]

    ; Free at scope exit
    Free v1

    Return
}
```

## Example 5: Nullable with Safe Call

**Source:**
```slang
Person = struct {
    val name: string
    val age: s64?
}

main = () {
    val p: Person? = Person{"Alice", 30}

    if p != null {
        print(p.age)
    }
}
```

**IR:**
```
function main() -> void {
b0:
    ; Create Person
    v0 = Const 24 [i64]         ; sizeof(Person) - nullable has 16 bytes
    v1 = Alloc v0 [*Person]

    ; Initialize name
    v2 = FieldPtr v1, 0 [*string]
    v3 = ConstStr "Alice" [string]
    Store v2, v3

    ; Initialize age (nullable s64 = 16 bytes)
    v4 = FieldPtr v1, 8 [*s64?]
    v5 = Const 30 [i64]
    v6 = Wrap v5 [s64?]         ; wrap in nullable
    Store v4, v6

    ; Wrap in nullable pointer (p: Person?)
    v7 = Wrap v1 [*Person?]

    ; if p != null
    v8 = IsNull v7 [bool]
    v9 = Not v8 [bool]
    Branch v9 -> b1, b2

b1:                             ; p is not null
    v10 = Unwrap v7 [*Person]   ; get underlying pointer
    v11 = FieldPtr v10, 8 [*s64?]
    v12 = Load v11 [s64?]
    v13 = Call print(v12) [void]
    Jump -> b2

b2:                             ; merge/exit
    Free v1
    Return
}
```

# Future Enhancements

1. **More optimization passes**: CSE, copy propagation, loop invariant code motion
2. **WebAssembly backend**: Portable execution target
3. **Debug info**: Source maps from IR to assembly
4. **Incremental compilation**: Only regenerate IR for changed functions
5. **IR serialization**: Save/load IR for caching and distribution
6. **Profile-guided optimization**: Use runtime data to optimize hot paths
