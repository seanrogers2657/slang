# Status

DRAFT, 2026-01-30

# Summary/Motivation

Add a module system to Slang enabling multi-file programs, code reuse, and namespace organization. Currently Slang is single-file only, which limits practical use for any non-trivial program. This proposal introduces file-based modules with explicit imports and directory-based packages, drawing primarily from Zig's import simplicity and Go's directory-as-package model.

Packages live in a dedicated `packages/` directory under the project root. This provides a clear boundary between importable packages and other project directories (docs, build artifacts, tests, etc.), simplifies the module resolver, and makes project structure self-evident at a glance.

Visibility modifiers (`private`, etc.) are deferred to a separate SEP. For now, all declarations are public.

# Goals/Non-Goals

- [goal] Multi-file compilation: programs can span multiple `.sl` files
- [goal] Explicit imports with clear dependency graph
- [goal] Namespace access via dot notation on imported modules
- [goal] No circular dependencies (enforced by compiler)
- [goal] Directory-based packages for grouping related files
- [goal] Self-evident project structure via `packages/` directory convention
- [non-goal] Visibility modifiers (`private`, `internal`, etc.) -- deferred to a separate SEP
- [non-goal] Standard library modules (deferred to future work; the stdlib will ship alongside the compiler binary when implemented)
- [non-goal] Dynamic linking or shared libraries
- [non-goal] Generics or parameterized modules
- [non-goal] Re-exporting (e.g., `pub import`)
- [non-goal] Wildcard imports (`import *`)
- [non-goal] Runtime module loading
- [non-goal] Nested module declarations within a file
- [non-goal] Versioned dependencies or a package manager

# Design Decisions

These decisions were made during proposal planning:

1. **Package = Directory**: A directory of `.sl` files forms a package. The directory path within `packages/` determines the package identity. Individual files cannot be imported -- only packages (directories). Files within the same directory are part of the same package and can access each other's declarations directly without imports. This follows Go's model and eliminates the ambiguity of file-vs-directory resolution.

2. **Everything is Public**: All top-level declarations are exported. Visibility modifiers are deferred to a separate SEP.

3. **Import as Assignment**: Imports use Slang's existing assignment syntax: `math = import("math")`. This is consistent with how Slang declares everything via assignment (`main = () { }`, `Point = struct { }`). `import` is a built-in function restricted to top-level assignments. The path must be a plain string literal -- no string concatenation, no interpolation, no variables. The path must be fully determinable from the source text alone, enabling the compiler to resolve all dependencies in a single pass before any evaluation. The import returns a package namespace accessed via dot notation. Note that `import` becomes a reserved keyword and can no longer be used as an identifier name. The binding name (left-hand side) is arbitrary and does not need to match the directory name -- e.g., `m = import("math")` is valid and the package is accessed as `m.add(1, 2)`.

4. **Imports Target Directories in `packages/`**: `import("math")` resolves to the `packages/math/` directory under the project root, never to a single file. This eliminates file-vs-directory ambiguity entirely. If you want a simple module, create a directory with one file in it. All declarations from all `.sl` files in the directory form a single namespace for the importer.

5. **No Circular Dependencies**: The compiler rejects circular import chains. This enforces clean architecture and simplifies compilation order. If A imports B, B cannot import A (directly or transitively).

6. **`packages/`-Relative Imports**: All import paths resolve within the `packages/` directory under the project root. The project root is the directory containing the entry file (the file passed to `sl build` or `sl run`). Paths do not use `./` or `../` prefixes. For example, `import("math")` resolves to `<project_root>/packages/math/`. `import("utils/helpers")` resolves to `<project_root>/packages/utils/helpers/`. In the future, standard library imports will use a reserved prefix (e.g., `import("std/math")`), but standard library support is deferred.

7. **Import at Top Level Only**: Import statements must appear at the top of a file, before any other declarations. This makes dependency scanning fast (no need to parse the whole file). Unused imports are allowed -- the compiler does not warn or error on them.

8. **No Implicit Prelude**: Nothing is implicitly imported. Built-in functions (`print`, `exit`, `len`, etc.) remain globally available as compiler intrinsics, not as imports. This keeps the current behavior unchanged.

9. **Package Initialization**: Top-level declarations (functions, structs, classes, objects, variables) are the only things allowed at the top level -- no bare statements. Variable initializers can call functions (e.g., `val config = load_config()`), and these execute at runtime before `main()`. Initialization order is deterministic:
    - **Across packages**: topological sort by import graph (dependencies initialize first)
    - **Across files within a package**: dependency graph of top-level declarations, with alphabetical filename as tiebreaker when there is no dependency relationship
    - **Within a file**: top-to-bottom source order
    - Circular initialization dependencies within a package (e.g., `val x = y + 1` in `a.sl` and `val y = x + 1` in `b.sl`) are a compile error

10. **Compilation Model**: `SlPackageCompiler` receives an explicit list of `.sl` files for the root package from its caller. The `sl` CLI discovers all `.sl` files in the entry file's directory and passes them in; tests pass just the single entry file. The compiler then discovers all imports transitively by scanning the `packages/` directory, topologically sorts packages, and compiles in dependency order. Each package is compiled once regardless of how many packages import it. The entry file can be located anywhere and the `sl` tool can be invoked from any directory. The directory containing the entry file is the root package -- it may contain additional `.sl` files that are part of the same package. The root package must contain a `main` function (compile error if missing). Only the specific file passed to `sl build` or `sl run` may define `main` -- other `.sl` files in the same root directory and all imported packages must not contain `main` functions (compile error: "package '<name>' must not declare a 'main' function"). If any package fails to compile, the build fails. Phase 1 collects all lexer, parser, and module errors before halting. Phase 2 stops at the first semantic error since later packages depend on earlier ones. All `.sl` files within a package directory are parsed before semantic analysis begins. The analyzer's two-pass approach (register all names, then type-check) enables forward references across files within the same package. Subdirectories within a package are not recursed into automatically -- they are independent packages that must be imported separately if needed (e.g., `import("math/integers")`).

11. **Name Conflicts**: If two imports would create the same binding name, the compiler reports an error. The user must alias one of them. Import binding names also participate in same-package duplicate name checking -- if an import binding has the same name as a declaration in a sibling file of the same package, this is a compile error.

12. **Global Variable Mutability**: Top-level `var` declarations are mutable and can be read and written by any package that imports them (e.g., `config.count = 5`). Top-level `val` declarations are read-only. Access control (restricting which packages can mutate a global) is deferred to the visibility SEP. Top-level `var` declarations with owned pointer types (`*T`) cannot be moved out of. Reading a global `*T` yields an implicit borrow (`&T` or `&&T` depending on context), never a move. This applies transitively -- if a global struct contains a `*T` field, that field also cannot be moved out of. To obtain an independent owned value, the caller must use `.copy()`. This prevents accidental invalidation of global state -- moving out of a global would leave it in an unusable state for the rest of the program. Primitive types (`s64`, `bool`, etc.) are unaffected since they are copied by value.

13. **Same-Package Access**: Within a directory package, files see each other's declarations directly using unqualified names (no imports needed). This follows Go's model. Duplicate names across files in the same package are a compile error.

14. **Root Package is Not Importable**: The root package (entry file's directory) exists outside of `packages/` and has no import path. It cannot be imported by any package. This is enforced structurally -- the package resolver only searches within `packages/`, so there is no path that could reference the root. This eliminates an entire class of circular dependency edge cases (any package imported by root could never import root back).

15. **Import Path Character Set**: Import path segments must match `[a-z][a-z0-9_]*`. Each segment between `/` separators is validated independently. Uppercase letters, hyphens, dots, and other special characters are rejected with a compile error. The following paths are reserved and cannot be used as package names: `main` (reserved for the root package's mangling prefix) and `std` (reserved for the future standard library). `import("main")` and any path starting with `std/` are compile errors. This ensures import paths are valid Slang-style identifiers, produce clean mangled assembly labels, and map predictably to filesystem directories.

16. **Unified Pipeline**: All programs — whether one file or many — go through the same compilation pipeline. A single-file program with no imports is the degenerate case: one root package, one file, zero dependencies. There is no separate "single-file mode." Name mangling, init function generation, and package resolution all apply uniformly. This eliminates branching in the compiler and ensures single-file programs are tested by the same code path as multi-file programs. The output changes slightly (e.g., `main` becomes `main_.main` in assembly), but behavior is identical.

17. **Nominal Typing Across Packages**: Types from different packages are distinct even if structurally identical. If `geometry` defines `Point` with `val x: s64, val y: s64` and `physics` defines `Vector` with `val x: s64, val y: s64`, these are different types. A function accepting `geometry.Point` will not accept a `physics.Vector`. This is a consequence of type identity being based on package path + declaration name (see Step 5).

18. **Transitive Type Exposure**: A function may return or accept types from one of its own dependencies. The caller can use such values without importing the origin package -- type inference, field access, method calls, and passing to other functions all work. However, the caller cannot *name* the type in annotations (e.g., `val p: geometry.Point = ...`) or *construct* instances (e.g., `geometry.Point{ 1, 2 }`) without importing the origin package. This matches Go's behavior: you can pass around values of types you didn't import, but you need the import to refer to the type by name.

# APIs

- `import("path")` - Built-in function that loads a package and returns its namespace. Path must be a string literal, resolved within the `packages/` directory under the project root. Restricted to top-level assignments.
- Package namespace access via `.` operator (e.g., `math.add(1, 2)`).

# Description

## Step 1: Lexer Changes

**File:** `compiler/lexer/lexer.go`

Add token support:

```go
TokenTypeImport   // 'import' keyword
```

Add to keywords map:
```go
"import":  TokenTypeImport,
```

No new operators needed. The existing `(`, `)`, `"`, and `.` tokens handle import syntax.

## Step 2: Parser Changes

**Files:** `compiler/ast/ast.go`, `compiler/parser/parser.go`

### AST Changes

Add `File` field to `Position` so that AST nodes carry their source file origin. The lexer already has the filename — it writes it into each token's position. The parser copies positions from tokens, so file info propagates to all AST nodes automatically:
```go
type Position struct {
    File   string  // source filename (new)
    Line   int
    Column int
    Offset int
}
```

Note: `ir.Position` already has a `File` field. This closes the gap between the AST and IR layers. The separate `errors.Position` type is left unchanged — `errors.CompilerError` continues using its own `Filename` field. When constructing errors from AST nodes, the `File` from `ast.Position` is copied into `CompilerError.Filename`. Unifying the two position types is deferred.

Add import declaration:
```go
type ImportDecl struct {
    Name     string   // binding name (e.g., "math")
    Path     string   // import path (e.g., "utils" or "std/math")
    Position Position
}
```

Add top-level variable declaration as a new `Declaration` type. Top-level variables have different semantics from in-function variables (global heap allocation, init ordering, package exports), so they get a dedicated AST node rather than reusing `VarDeclStmt`:
```go
type TopLevelVarDecl struct {
    Name          string     // variable name
    Mutable       bool       // true for var, false for val
    TypeAnnotation string    // explicit type annotation (e.g., "s64", "geo.Point"), empty if inferred
    Value         Expression // initializer expression
    StartPos      Position
    EndPos        Position
}
```

Add `Imports` field to the existing `Program`:
```go
type Program struct {
    Imports      []*ImportDecl      // import declarations (new)
    Declarations []Declaration      // all other declarations (existing, plus TopLevelVarDecl)
    // ... existing fields ...
}
```

### Parser Changes

**Parse import declarations**: An import looks like `name = import("path")`. The parser recognizes this pattern when it sees an identifier followed by `=` followed by the `import` keyword.

```
math = import("math")
```

Parses as:
- Name: `"math"`
- Path: `"math"`

**Qualified type names in annotations**: The parser must accept qualified names (`pkg.Type`) anywhere a type is currently accepted. This includes nullable types (`pkg.Type?`), pointer types (`*pkg.Type`), and borrow types (`&pkg.Type`, `&&pkg.Type`). The existing `parseTypeName()` returns a string (e.g., `"*Point"`, `"s64?"`). For qualified types, after consuming the type identifier, the parser checks for `.` followed by another identifier and produces a dotted string (e.g., `"geo.Point"`). No new AST node is needed — the semantic analyzer splits on `.` to resolve the package namespace and type name.

```go
// In parseTypeName(), after consuming the type identifier:
// Check for qualified type: pkg.Type (e.g., geo.Point)
if p.CurrentToken().Type == lexer.TokenTypeDot {
    p.advance() // consume '.'
    if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
        p.addError(...)
        return "", typePos
    }
    typeName = typeName + "." + p.CurrentToken().Value
    p.advance() // consume qualified name
}
```

This naturally composes with existing modifiers: `*geo.Point` parses as `"*" + parseTypeName()` which produces `"*geo.Point"`. Same for `&geo.Point`, `&&geo.Point`, and `geo.Point?`.

```slang
geo = import("geometry")

transform = (p: &geo.Point) -> *geo.Point {
    val result: geo.Point? = null
    // ...
}
```

**Qualified struct literal construction**: The parser must handle `geo.Point{ 1, 2 }` and `geo.Point { 1, 2 }` as struct literal construction with a package qualifier. When the parser is in the dot-access code path and sees `{` after the member name, it parses this as a qualified struct literal. Unlike unqualified struct literals (which require no space before `{` to avoid ambiguity with control flow like `if x {`), qualified struct literals allow whitespace before `{` because `geo.Point {` is unambiguous — a dot expression cannot appear as a control flow condition.

The `StructLiteral` AST node gains optional `PackageAlias` and `PackageAliasPos` fields:

```go
type StructLiteral struct {
    PackageAlias    string          // import alias (e.g., "geo"), empty if unqualified
    PackageAliasPos Position        // position of package alias, zero if unqualified
    Name            string          // struct name (e.g., "Point")
    NamePos         Position        // position of struct name
    LeftBrace       Position        // position of '{'
    Arguments       []Expression    // list of positional argument expressions
    NamedArguments  []NamedArgument // list of named arguments (e.g., x: 10, y: 20)
    RightBrace      Position        // position of '}'
}
```

In the parser's dot-access path (after consuming `memberName`), a new check is added before the field access fallthrough:

```go
// After consuming memberName in dot-access path:

// Check if this is a method call (identifier followed by '(')
if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeLParen {
    // ... existing method call parsing ...
}

// Check if this is a qualified struct literal (pkg.Type{ ... } or pkg.Type { ... })
// Whitespace before '{' is allowed for qualified literals (unambiguous)
if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeLBrace {
    // left must be a simple IdentifierExpr (the package alias)
    if ident, ok := left.(*ast.IdentifierExpr); ok {
        return p.parseStructLiteral(memberName, memberPos, ident.Name, ident.StartPos)
    }
}

// Otherwise it's a field access
```

The semantic analyzer validates that `PackageAlias` (when non-empty) refers to a `SlPackageNamespace` and that `Name` is a struct/class type in that package.

## Step 3: Package Resolution

**New file:** `compiler/slpackage/resolver.go`

The `SlPackageResolver` (defined in Step 4) translates import paths to file system paths within the `packages/` directory.

Resolution rules:
- `import("foo")` -> look for `<RootDir>/packages/foo/` directory (must be a directory)
- `import("foo/bar")` -> look for `<RootDir>/packages/foo/bar/` directory
- Paths must not start with `./`, `../`, or `/` -- these are compile errors
- Path segments must match `[a-z][a-z0-9_]*` -- invalid characters are a compile error
- `import("main")` is a compile error -- `main` is reserved for the root package's mangling prefix
- `import("std")` and paths starting with `std/` are compile errors -- reserved for the future standard library
- If `packages/` does not exist and imports are present, emit an error: "no 'packages' directory found; create a 'packages/' directory in the project root to use imports"
- If `packages` exists but is not a directory, emit an error: "'packages' exists but is not a directory"
- If the path does not resolve to a directory within `packages/`, emit an error
- If the directory exists but contains no `.sl` files, emit an error (e.g., "package 'math' has no .sl files")
- If `packages/` contains loose `.sl` files (not inside a subdirectory), emit a warning: "file 'helpers.sl' is directly in 'packages/' and is not part of any package; move it into a subdirectory"
- Standard library paths (e.g., `import("std/math")`) are reserved for future use

## Step 4: Compilation Pipeline

**File:** `cmd/sl/main.go` and new `compiler/slpackage/compiler.go`

The compilation pipeline is unified — all programs go through the same phases. Here is the full pipeline with package support:

```
sl build main.sl / sl run main.sl
    │
    ▼
┌─────────────────────────────────────────────┐
│  Phase 1: Discovery & Parsing               │
│  (compiler/slpackage, compiler/lexer,       │
│   compiler/parser)                          │
│                                             │
│  Starting from the entry file's directory:  │
│  1. Read + lex + parse all .sl files        │
│  2. Group per-file ASTs into PackageAST     │
│  3. Extract imports from parsed ASTs        │
│  4. Resolve import paths within packages/   │
│  5. For each discovered package, repeat 1-4 │
│  6. Cycle detection (DFS on import graph)   │
│  7. Topological sort for later phases       │
│                                             │
│  Returns: map[string]*PackageAST            │
│  (per-file ASTs grouped by package path)    │
│                                             │
│  A single-file program with no imports      │
│  produces one PackageAST with one FileAST   │
│                                             │
│  Errors: lexer/parser errors, missing       │
│  package, empty package, invalid path,      │
│  circular dependency                        │
│  Error stages: "lexer", "parser", "module"  │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│  Phase 2: Semantic Analysis                 │
│  (compiler/semantic)                        │
│                                             │
│  Consumes PackageASTs from Phase 1.         │
│  For each package (in topological order):   │
│    1. Register all names from all files     │
│    2. Bind imports to SlPackageNamespaces   │
│    3. Type check all file bodies            │
│    4. Populate SlPackage.TypedAST & Exports │
│    5. Detect circular init dependencies     │
│                                             │
│  Errors: type errors, undefined references, │
│  init cycles                                │
│  Error stage: "semantic"                    │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│  Phase 3: IR Generation                     │
│  (compiler/ir)                              │
│                                             │
│  For each package:                          │
│    Generate SSA IR with mangled names       │
│  Combine into single *ir.Program            │
│  Generate package init functions            │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│  Phase 4: ARM64 Backend                     │
│  (compiler/ir/backend/arm64)                │
│                                             │
│  Emit single .s file from combined IR       │
│  Assemble → link → executable               │
│  (sl run: execute the binary)               │
└─────────────────────────────────────────────┘
```

The pipeline steps in detail:

1. **Discovery & Parsing**: The compiler receives an explicit list of `.sl` files for the root package (provided by the caller). It reads, lexes, and parses these files, grouping per-file ASTs into a `PackageAST` (file boundaries are preserved, not merged). It then extracts imports from the parsed ASTs, resolves paths within `packages/`, and recursively discovers and parses transitive dependencies. For imported packages (within `packages/`), all `.sl` files in the directory are always discovered automatically. Each file is read and parsed exactly once -- parsing order is irrelevant since parsing is purely syntactic. After all packages are discovered, run cycle detection (DFS) and compute the topological sort. Phase 1 returns a `map[string]*PackageAST` for Phase 2 to consume. For a single-file program with no imports, this produces one `PackageAST` containing one `FileAST`.
2. **Semantic Analysis**: Consume the `PackageAST` values from Phase 1. Analyze each package in topological (dependency) order using a two-pass approach: first register all top-level names from all files, then type-check all bodies. For each package, bind imports to `SlPackageNamespace` values from already-analyzed dependencies, and populate `SlPackage.TypedAST` and `SlPackage.Exports`. The `PackageAST` values are discarded after this phase.
3. **IR Generation**: Generate IR for each package from its `TypedAST` with mangled names. Combine into a single `*ir.Program`.
4. **Code Generation**: Emit combined assembly from the single `*ir.Program`, assemble, and link.

All programs — including single-file programs with no imports — go through this pipeline. A single-file program simply produces one root package with one `FileAST` and an empty dependency graph. Since parsing is purely syntactic (no cross-file or cross-package dependencies), parse order does not matter. The topological sort computed in Phase 1 only governs Phase 2 onward, where dependency order is required.

**Root package file discovery**: `SlPackageCompiler` does not discover root package files itself — it receives an explicit list from its caller. The `sl` CLI discovers all `.sl` files in the entry file's directory and passes them in. Tests pass just the single entry file. This separation keeps the compiler testable without filesystem side effects and avoids sibling file conflicts in test directories. Imported packages (within `packages/`) always have all their `.sl` files discovered automatically by the compiler, since they are self-contained directories.

This introduces a new error stage `"module"` for errors during Phase 1 (import resolution, circular dependencies, missing/empty packages). Lexer and parser errors are also reported during Phase 1, since parsing now happens in this phase.

### Entry Point Restructuring

The current `compileSourceWithIR` function in `cmd/sl/main.go` calls lexer, parser, analyzer, IR generator, and ARM64 backend sequentially for a single file. With the package system, this function delegates to `SlPackageCompiler` for Phases 1–3, then passes the combined IR to the backend:

```go
func compileSourceWithIR(filename string, verbose bool, timer *timing.Timer) (string, error) {
    // Determine project root (directory containing the entry file)
    rootDir := filepath.Dir(filename)

    // Discover all .sl files in the root package directory
    rootFiles := discoverSlFiles(rootDir)

    // Phase 1: Discovery & Parsing
    compiler := slpackage.NewCompiler(rootDir, filename, rootFiles)
    pkgASTs, err := compiler.DiscoverAndParse()
    if err != nil {
        return "", err  // lexer, parser, or module errors
    }

    // Phase 2: Semantic Analysis
    if err := compiler.Analyze(pkgASTs); err != nil {
        return "", err  // type errors
    }

    // Phase 3: IR Generation
    // Iterate packages in topological order, generate IR with mangled names,
    // combine into a single *ir.Program
    irProg, err := compiler.GenerateIR()
    if err != nil {
        return "", err
    }

    // Validate IR
    if irErrors := ir.Validate(irProg); len(irErrors) > 0 {
        return "", fmt.Errorf("IR validation failed")
    }

    // Phase 4: ARM64 Backend (unchanged — operates on a single *ir.Program)
    arm64Backend := arm64.New(&backend.Config{Filename: filename})
    return arm64Backend.Generate(irProg)
}
```

The key structural change: `compileSourceWithIR` no longer reads source files, calls the lexer, or calls the parser directly. All of that moves into `SlPackageCompiler.DiscoverAndParse()`. The function becomes a thin orchestrator that constructs the compiler, runs each phase, and hands the combined IR to the backend.

**Error collection strategy**: Phase 1 collects all lexer, parser, and module errors across all discovered packages before halting. If a package has a parse error, the compiler does not discover that package's imports (since the AST is incomplete), but continues parsing other already-discovered packages. All collected errors are reported together. Phase 2 (semantic analysis) stops at the first package with a semantic error, since later packages in the topological order depend on earlier ones being valid.

### Type Definitions

Eight new types and two updated types support the package system. Each has a single, well-defined role:

```go
// --- AST layer (compiler/ast/ast.go) ---

// ImportDecl represents `math = import("math")` in the AST.
type ImportDecl struct {
    Name     string   // binding name (e.g., "math")
    Path     string   // import path (e.g., "math", "utils/helpers")
    Position Position
}

// Position gains a File field for multi-file error reporting.
// The lexer sets File from its constructor argument; the parser
// copies positions from tokens, so file info propagates automatically.
type Position struct {
    File   string  // source filename
    Line   int
    Column int
    Offset int
}

// Program gains an Imports field. Imports are separated from Declarations
// so Phase 1 can extract them without walking all declarations.
type Program struct {
    Imports      []*ImportDecl   // import declarations (new)
    Declarations []Declaration   // all other declarations (existing)
    // ... existing fields ...
}
```

```go
// --- Package layer (compiler/slpackage/) ---

// FileAST pairs a source file path with its parsed AST.
// File boundaries are preserved (not merged) so that error messages
// can report which file an error came from.
type FileAST struct {
    Path string        // e.g., "packages/utils/format.sl"
    AST  *ast.Program  // parsed AST for this one file
}

// PackageAST groups the per-file ASTs for all files in a package.
// This is the output of Phase 1 and the input to Phase 2.
// A single-file program produces a PackageAST with one FileAST.
type PackageAST struct {
    Files []*FileAST  // one per .sl file, in alphabetical order
}

// SlPackageResolver translates import paths to filesystem directories.
// Used during Phase 1 to locate packages within the packages/ directory.
type SlPackageResolver struct {
    RootDir       string            // project root (entry file's directory)
    PackagesDir   string            // RootDir + "/packages"
    ResolvedPaths map[string]string // import path -> absolute directory path
}

// SlPackageCompiler orchestrates compilation across all packages.
// It owns all discovered packages and their compilation order.
// The root package files are provided explicitly by the caller;
// imported packages are discovered automatically from packages/.
type SlPackageCompiler struct {
    RootDir      string                // project root directory
    EntryFile    string                // the specific file passed to sl build/run (must contain main)
    RootFiles    []string              // all .sl files for root package (includes EntryFile)
    Resolver     *SlPackageResolver
    Packages     map[string]*SlPackage
    CompileOrder []string              // topological order of package paths
}

// SlPackage represents a single compilation unit (one directory of .sl files).
// Identity fields (Path, Dir) are set at creation during Phase 1.
// Result fields (TypedAST, Exports) are populated during Phase 2.
type SlPackage struct {
    // Identity
    Path string // import path ("math", "utils/helpers"; "main" for root)
    Dir  string // absolute directory path on disk

    // Phase 2 results
    TypedAST *semantic.TypedProgram    // type-checked program (one per package)
    Exports  map[string]semantic.Export // public symbols, keyed by name
}

```

```go
// --- Semantic layer (compiler/semantic/) ---

// Export represents a single public symbol from a package.
// The symbol's kind (function, struct, class, variable) is encoded in the
// Type field -- no separate kind enum is needed.
// Defined in semantic to avoid circular imports -- slpackage references
// semantic.Export, while semantic has no dependency on slpackage.
type Export struct {
    Type    Type // the symbol's type (FunctionType, StructType, etc.)
    Mutable bool // true for `var` declarations; false otherwise
}

// SlPackageNamespace is bound to an import alias in the analyzer's scope.
// When the analyzer processes `math = import("math")`, it creates a
// SlPackageNamespace and binds it to "math" in the current scope.
type SlPackageNamespace struct {
    Path    string            // canonical import path (e.g., "math")
    Exports map[string]Export // references SlPackage.Exports directly
}
```

**Phase 1 returns `PackageAST` values** -- they are not stored on `SlPackage`:

```go
func (c *SlPackageCompiler) DiscoverAndParse() (map[string]*PackageAST, error) {
    // Root package: parse the explicit files in c.RootFiles
    // Imported packages: discover all .sl files in the resolved directory
    // For each package:
    //   1. Parse each file into *ast.Program
    //   2. Group into PackageAST (file boundaries preserved)
    //   3. Extract imports from parsed ASTs, resolve paths, discover transitive deps
    //   4. Cycle detection and topological sort
    // Populates c.Packages (identity fields) and c.CompileOrder
    // Returns PackageASTs keyed by package path
}
```

**Phase 2 consumes the PackageASTs and populates results on SlPackage**:

```go
func (c *SlPackageCompiler) Analyze(pkgASTs map[string]*PackageAST) error {
    for _, path := range c.CompileOrder {
        pkg := c.Packages[path]
        pkgAST := pkgASTs[path]
        // Two-pass analysis:
        //   Pass 1: register all top-level names from all files
        //   Pass 2: type-check all file bodies (all names now known)
        // Create SlPackageNamespace for each import from already-analyzed deps
        pkg.TypedAST = analyzePackage(pkgAST, /* dependency namespaces */)
        pkg.Exports = extractExports(pkg.TypedAST)
    }
    // After this, the pkgASTs map can be garbage collected
}
```

## Step 5: Semantic Analysis Changes

**File:** `compiler/semantic/analyzer.go`

### Two-Pass Analysis

The analyzer gains a new entry point that accepts a `PackageAST` instead of a single `*ast.Program`:

```go
func (a *Analyzer) AnalyzePackage(
    pkg *PackageAST,
    deps map[string]*SlPackageNamespace,
) ([]*errors.CompilerError, *TypedProgram)
```

Analysis uses two conceptual phases to support cross-file forward references:

**Phase A — Registration**: Walk all files and register every top-level name (functions, structs, classes, imports, variables) into the package-level symbol table. After this phase, all names are known but no bodies have been type-checked. This is what enables file A to call a function defined in file B. The existing analyzer already uses a multi-pass registration strategy (register type names, resolve fields/methods, collect function signatures). This internal structure can be preserved or simplified as needed — the important invariant is that all names are fully registered before type-checking begins.

**Phase B — Type checking**: Walk all files again and type-check every declaration body. Because all names were registered in Phase A, forward references resolve normally. Each file's declarations are checked in source order; files are processed in alphabetical order.

`AnalyzePackage` is the sole entry point for semantic analysis. The old `Analyze(*ast.Program)` method is removed. Existing unit tests are updated to construct a one-file `PackageAST` and call `AnalyzePackage` directly. This ensures all code paths — tests included — exercise the same logic.

**`main` function validation**: The `SlPackageCompiler` tracks the entry file (the file passed to `sl build`/`sl run`). During Phase 2, the root package is validated:
- The entry file must define a `main` function (compile error if missing)
- Other root package files must not define `main` (compile error: "'main' must be defined in the entry file '<entry>.sl', not in '<other>.sl'")
- Imported packages must not define `main` (compile error: "package '<name>' must not declare a 'main' function")

The output is one flat `*TypedProgram` per package. File boundaries are not needed after semantic analysis — each typed node carries its source position (including the `File` field), and all names are resolved.

### Package-Aware Symbol Table

The analyzer uses `SlPackageNamespace` (defined in Step 4) to represent imported packages in scope.

When analyzing an import declaration (e.g., `math = import("math")`):
1. Look up `"math"` in the `deps` map passed to `AnalyzePackage`
2. Bind the `SlPackageNamespace` to `"math"` in the current scope

When analyzing a dot expression on a namespace (e.g., `math.add`):
1. Check if the left side resolves to a `SlPackageNamespace`
2. Look up the right side in `ns.Exports`
3. If not found, produce a clear error (e.g., "package 'math' has no declaration 'foo'")
4. Return the `Export.Type` of the symbol

### Method Dispatch on Imported Types

When a method is called on an instance of an imported type (e.g., `a.get_balance()` where `a` has type `account.Account`), the analyzer resolves the method through the type's canonical package identity:

1. Determine the receiver type (e.g., `account.Account`)
2. Look up the type's origin package using the canonical package path (not the import alias)
3. Search for the method in that package's class/struct definition
4. Validate the method signature (self parameter, argument types, return type)

This means method resolution follows the type, not the import alias. If `acct = import("account")` and `a2 = import("account")`, then `acct.Account` and `a2.Account` are the same type and both resolve methods from the `account` package.

### Qualified Type Names in Annotations

The analyzer must resolve qualified type names in type annotations. When it encounters `geo.Point` in a type position (e.g., function parameter, variable type annotation, nullable type), it:

1. Looks up `geo` in the current scope -- expects a `SlPackageNamespace`
2. Looks up `Point` in `ns.Exports` -- expects a type (struct, class)
3. Returns the canonical type identity using `ns.Path` + `"Point"` (e.g., `geometry.Point`)

This applies to all type positions: `&geo.Point`, `&&geo.Point`, `*geo.Point`, `geo.Point?`, and array element types.

### Cross-Package Type Identity

Types are identified by their **package path + declaration name**, not by the import alias used. For example, if `geometry/point.sl` declares `Point`, the canonical type identity is `geometry.Point` regardless of how it is imported:

```slang
geo = import("geometry")
g = import("geometry")

val a = geo.Point{ 1, 2 }
val b = g.Point{ 3, 4 }
// a and b have the same type: geometry.Point
```

This means two imports of the same package under different aliases produce compatible types, and functions accepting `geometry.Point` will accept values created through any alias.

#### Implementation: `PackagePath` field on nominal types

The three nominal types in the semantic layer (`StructType`, `ClassType`, `ObjectType`) each gain a `PackagePath` field. Type equality checks both `Name` and `PackagePath`:

```go
type StructType struct {
    Name        string            // "Point"
    PackagePath string            // "geometry" (or "main" for root package)
    Fields      []StructFieldInfo
}

func (t StructType) Equals(other Type) bool {
    o, ok := other.(StructType)
    if !ok {
        return false
    }
    return t.Name == o.Name && t.PackagePath == o.PackagePath
}

func (t StructType) String() string {
    if t.PackagePath == "" || t.PackagePath == "main" {
        return t.Name
    }
    return t.PackagePath + "." + t.Name
}
```

The same pattern applies to `ClassType` and `ObjectType`. `FunctionType` is structural (no name), so it needs no change.

**How `PackagePath` is set:**
- During semantic analysis Pass 1, the analyzer registers types with the `PackagePath` of the package being analyzed (e.g., `"geometry"` for types in `packages/geometry/`). The root package uses `"main"`.
- When resolving `geo.Point`, the analyzer looks up `Point` in the `SlPackageNamespace` for `"geometry"`, which already has `PackagePath: "geometry"` set.

**Wrapper types work automatically:** `OwnedPointerType`, `RefPointerType`, `MutRefPointerType`, `NullableType`, and `ArrayType` all delegate equality to their inner/element type's `Equals()` method. No changes needed for these.

**IR layer:** The `ir.StructType` uses mangled names (e.g., `geometry_.Point`), which are globally unique. No extra `PackagePath` field is needed in the IR.

## Step 6: IR Generator Changes

**File:** `compiler/ir/generator.go`

The IR generator iterates through packages in `CompileOrder`, generating IR from each package's `TypedAST` into a single combined `*ir.Program`. A single `Generator` instance and single `*Program` are shared across all packages:

```go
func (c *SlPackageCompiler) GenerateIR() (*ir.Program, error) {
    g := ir.NewGenerator()

    for _, path := range c.CompileOrder {
        pkg := c.Packages[path]
        g.SetPackagePath(path)  // controls name mangling for this package
        if err := g.GeneratePackage(pkg.TypedAST); err != nil {
            return nil, err
        }
    }

    // Generate init functions for packages with top-level variables
    for _, path := range c.CompileOrder {
        pkg := c.Packages[path]
        g.GenerateInitFunction(path, pkg.TypedAST)
    }

    return g.Program(), nil
}
```

### Name Mangling

All names are mangled with the package path as prefix:

```go
func mangleName(packagePath string, name string) string {
    // Convert "math" + "add" to "math_.add"
    // Convert "utils/helpers" + "format" to "utils_.helpers_.format"
    // Root package: "main" + "my_func" to "main_.my_func"
    // Uses _. as separator -- unambiguous since . cannot appear in Slang identifiers
    // and both _ and . are valid in macOS ARM64 assembly labels
}
```

### How Multiple Packages Combine

**Functions flatten into one `Functions` slice.** Mangled names prevent collisions. `math_.add` and `main_.add` coexist in the same slice.

**Struct types use mangled names.** `geometry_.Point` and `physics_.Vector` are distinct IR structs. Each package registers its own structs during its generation pass. The `Generator.typeCache` (`semantic.Type` → `ir.Type`) ensures that if multiple packages reference `geometry.Point`, the same `ir.StructType` is reused — the `PackagePath` field on `semantic.StructType` makes each semantic type a unique cache key.

**String constants deduplicate automatically.** `Program.AddString()` already uses a `stringIndex` map. If package A and package B both use the string `"hello"`, they get the same index.

**Global variables flatten into one `Globals` slice.** Mangled names prevent collisions (e.g., `math_.pi`, `config_.db_port`).

**Cross-package calls resolve by mangled name.** When package B calls `math.add(1, 2)`, the semantic layer has already resolved this to the `add` function in the `math` package. The IR generator emits a call to the mangled name `math_.add`.

### Generator State

The `Generator` gains a `packagePath` field set via `SetPackagePath(path)`. This field is used by `registerStruct`, `generateFunction`, `generateClass`, and `generateObject` to produce mangled names. All other generator state (SSA builder, type cache, program) persists across packages.

## Step 7: ARM64 Backend Changes

**File:** `compiler/ir/backend/arm64/backend.go`

Minimal changes needed:
- Function labels use mangled names
- Cross-package calls use `bl` to mangled labels
- All package code is emitted into a single assembly file (no separate object files per package)
- `_start` reads `ir.Program.InitOrder` to emit `bl` calls to init functions before `main_.main`

The `ir.Program` gains an `InitOrder` field populated by `SlPackageCompiler` during IR generation:

**`ir.Program.Main()` and `Validate()` updates**: With the unified pipeline, `main` is always mangled to `main_.main`. `Program.Main()` looks for `"main_.main"` and `Validate()` checks for the same. No conditional logic — all programs go through mangling.

```go
type Program struct {
    Functions []*Function
    Structs   []*StructType
    Globals   []*Global
    Strings   []string
    InitOrder []string  // ordered init function names (e.g., ["logger_.init", "main_.init"])
    // ...
}
```

The backend emits `_start` as:
```asm
_start:
    // call init functions in dependency order
    bl _logger_.init       // from InitOrder[0]
    bl _main_.init         // from InitOrder[1]
    // then call main
    bl _main_.main
    mov x16, #1
    svc #0
```

Packages with no top-level variable initializers are omitted from `InitOrder` (no init function generated).

### Reserved `_sl_` Prefix for Internal Labels

All compiler-generated assembly labels use the `_sl_` prefix to avoid collisions with user-defined symbols. This applies to:

- Heap management: `_sl_heap_ptr`, `_sl_heap_end`, `_sl_arena_head`, `_sl_current_arena`, `_sl_free_lists`, `_sl_heap_alloc`
- Print helpers: `_sl_print_s64`, `_sl_print_string`, `_sl_print_bool`, `_sl_newline`, `_sl_true_str`, `_sl_false_str`
- Panic helpers: `_sl_panic`, `_sl_panic_div_zero`, `_sl_panic_mod_zero`, `_sl_panic_bounds`, etc.
- Assertion support: `_sl_assert_prefix`
- Entry point: `_start` (reserved by the system, not user-accessible)

Since import path segments must match `[a-z][a-z0-9_]*` and user mangled names use the `<pkg>_.` pattern (e.g., `math_.add`), the `_sl_` prefix is unambiguous — no valid package path starts with `_`.

## Step 8: Directory Packages

When an import path resolves to a directory within `packages/`:

1. Find all `.sl` files in the directory (alphabetical order)
2. Parse each file into its own `*ast.Program`
3. Group into a `PackageAST` with one `FileAST` per file (file boundaries preserved)
4. The semantic analyzer's two-pass approach treats all files as a single namespace

```
project/
  main.sl
  packages/
    utils/
      format.sl       ← one FileAST in the "utils" PackageAST
      convert.sl      ← one FileAST in the "utils" PackageAST
      internal/       ← NOT auto-included; separate package "utils/internal"
```

Rules:
- No subdirectory recursion (only direct `.sl` files)
- Files are always enumerated in alphabetical order for deterministic builds, error output, and initialization ordering
- Duplicate names across files in the same directory are a compile error
- Files within the same package can reference each other's declarations directly (no import needed)

## Step 9: Package Initialization

Top-level variable declarations can have initializers that call functions. These run at runtime, before `main()`, in a deterministic order.

### What is Allowed at the Top Level

Only declarations are allowed at the top level of a file:
- Function declarations: `add = (a: s64, b: s64) -> s64 { ... }`
- Struct declarations: `Point = struct { ... }`
- Class declarations: `Counter = class { ... }`
- Object declarations: `Math = object { ... }`
- Variable declarations: `val config = load_config()` or `var count: s64 = 0`
- Import declarations: `math = import("math")`

Bare statements (e.g., `print("hello")` outside any function) are **not allowed** at the top level.

### Initialization Order

Initialization proceeds in three tiers:

**1. Across packages** -- topological sort by import graph:
```
main imports validator, validator imports logger
→ logger initializes first, then validator, then main's top-level, then main()
```

**2. Across files within a package** -- dependency graph with alphabetical filename tiebreaker:

The compiler builds a dependency graph of top-level declarations across files. If `convert.sl` declares `val hex_table = build_table()` and `format.sl` declares `val fmt = use_hex(hex_table)`, then `hex_table` initializes before `fmt`. When two declarations have no dependency relationship, the file whose name comes first alphabetically initializes first.

```
packages/utils/
  alpha.sl    <- initializes before beta.sl (alphabetical tiebreaker)
  beta.sl
```

**3. Within a file** -- top-to-bottom source order:
```slang
val a = 1           // first
val b = a + 1       // second
val c = compute(b)  // third
```

### Circular Initialization

Circular dependencies between top-level declarations within a package are a compile error:

```slang
// a.sl
val x = y + 1   // depends on y from b.sl

// b.sl
val y = x + 1   // depends on x from a.sl
// Error: circular initialization dependency: x (a.sl) -> y (b.sl) -> x (a.sl)
```

### Global Variable Assembly Representation

All top-level variables are uniformly represented as **pointers to heap memory**. Each global gets a `.quad 0` slot in the `.data` section (holding a pointer), and the package init function allocates heap memory, initializes the value, and stores the pointer into the `.data` slot. This applies to all types — primitives (`s64`, `bool`), structs, arrays, and owned pointers alike.

**Access pattern:**
- **Reading a global**: load pointer from `.data` label, then load value through the pointer
- **Writing a `var` global**: load pointer from `.data` label, then store value through the pointer
- **Cross-package access** uses the same pattern with the mangled label (e.g., `math_.count`)

**Constant initializer example** (`val max_size: s64 = 42`):
```asm
// .data section
main_.max_size:
    .quad 0                                    // pointer slot, zero until init

// In _main_.init:
    mov x0, #8                                 // size of s64
    bl _sl_heap_alloc                           // returns heap pointer in x0
    mov x1, #42                                // initial value
    str x1, [x0]                               // store 42 into heap
    adrp x2, main_.max_size@PAGE
    str x0, [x2, main_.max_size@PAGEOFF]       // store pointer into .data slot
```

**Runtime initializer example** (`val config = load_config()`):
```asm
// In _main_.init:
    bl _main_.load_config                       // call function, result in x0
    mov x10, x0                                // save result
    mov x0, #8                                 // size of return type
    bl _sl_heap_alloc                           // allocate heap slot
    str x10, [x0]                              // store result into heap
    adrp x2, main_.config@PAGE
    str x0, [x2, main_.config@PAGEOFF]         // store pointer into .data slot
```

**Struct initializer example** (`val origin = Point{ 0, 0 }`):
```asm
// In _main_.init:
    mov x0, #16                                // size of Point (2 x s64)
    bl _sl_heap_alloc                           // allocate heap memory
    str xzr, [x0]                              // x field = 0
    str xzr, [x0, #8]                          // y field = 0
    adrp x2, main_.origin@PAGE
    str x0, [x2, main_.origin@PAGEOFF]         // store pointer into .data slot
```

### Generated Code

The compiler generates a package init function for each package that has top-level variable initializers. The entry point calls all init functions in dependency order before calling `main()`.

```asm
_start:
    bl _logger_.init      // dependency initializes first
    bl _validator_.init   // then its dependent
    bl _main_.init        // entry package last
    bl _main_.main        // then main()
    mov x16, #1
    svc #0
```

# Alternatives

1. **Rust-Style Explicit Module Tree**: Requiring `mod` declarations to build a module tree. Rejected as too complex for Slang's philosophy. Files should be automatically discovered, not manually declared.

2. **Single-File Imports**: Allowing `import("math")` to resolve to `math.sl` (a single file). Rejected because it creates ambiguity when both `math.sl` and `math/` exist. Directory-only imports are simpler and unambiguous. A single-file module is just a directory with one file in it.

3. **File-Relative Import Paths**: Resolving import paths relative to the importing file's directory (like Node.js/Python), using `./` and `../` prefixes. Rejected because it leads to fragile `../` chains for sibling packages and paths that change when files are moved. Root-relative paths are stable regardless of file location.

4. **C/C++ Header Files**: Separate declaration (`.h`) and implementation files. Rejected as unnecessarily complex and error-prone (declaration duplication).

5. **Kotlin-Style Package Declaration**: Having a `package` statement at the top of each file. Rejected because the file system path already determines the package. Adding a redundant declaration creates a source of errors when files are moved.

6. **Bare Top-Level Statements**: Allowing `print("hello")` at the top level outside any function. Rejected because it conflates declarations with imperative code, and makes initialization ordering harder to reason about. Use a function called from `main()` or from a variable initializer instead.

7. **Flat Package Layout (no `packages/` directory)**: Resolving import paths directly from the project root (e.g., `import("math")` -> `<root>/math/`). Rejected because it mixes importable packages with non-package directories (`docs/`, `build/`, `test/`, `.git/`), makes project structure ambiguous to humans, complicates the resolver (must distinguish packages from non-packages by context), and creates edge cases around root package importability. The `packages/` directory provides a clear boundary with no ambiguity.

# Testing

- **Lexer tests**: Token recognition for `import` keyword
- **Parser tests**:
  - Import declaration parsing
  - Qualified type names in annotations (`geo.Point`, `&geo.Point`, `geo.Point?`)
  - Error on import not at top level
- **Module resolution and discovery tests** (error stage: `module`):
  - Resolution within `packages/` directory
  - Rejection of `./`, `../`, and `/` prefixes
  - Rejection of invalid path characters (uppercase, hyphens, dots)
  - Directory package detection
  - Missing `packages/` directory error (when imports are present)
  - Missing package error
  - Empty directory error (no `.sl` files)
  - Warning for loose `.sl` files directly in `packages/`
  - Circular dependency detection with clear cycle path in error
- **Semantic tests**:
  - Cross-package type checking
  - Package namespace dot access
  - Method dispatch on imported class/struct instances
  - Qualified type names in function parameters and return types
  - Accessing declarations from imported packages
  - Duplicate name detection across files in the same package
  - Import binding conflicts with same-package declarations
  - Name conflict detection (two imports with same binding name)
  - Undefined import symbol error
  - Nominal typing: structurally identical types from different packages are distinct
- **IR tests**:
  - Name mangling correctness
  - Cross-package function calls
- **Initialization tests**:
  - Package init order follows import dependency graph
  - Intra-package init order: dependency graph with alphabetical filename tiebreaker
  - Circular initialization dependency detection
  - Top-level `val` with function call initializer
  - Top-level `var` with initializer
  - Error on bare statements at top level
- **Transitive type exposure tests**:
  - Caller can use values of types it didn't import (field access, method calls, passing)
  - Caller cannot name an unimported type in annotations (compile error)
  - Caller cannot construct instances of an unimported type (compile error)
- **Single-file program tests** (unified pipeline):
  - Single-file program with no imports compiles correctly through the package pipeline
  - Single-file program produces mangled names (`main_.main`) in assembly output
  - Two `.sl` files in root directory are treated as one root package
- **E2E tests**:
  - All E2E tests use `SlPackageCompiler` (unified pipeline) -- there is no separate single-file mode
  - Existing single-file tests in `_examples/slang/` pass one file to `SlPackageCompiler` as the root package
  - Multi-file project tests live in `_examples/slang/projects/`, each as a directory containing `main.sl` and optionally `packages/`
  - `@test:` directives are read from `main.sl` only (the entry file)
  - `error_contains` is sufficient for asserting error messages from any file in the project
  - Test discovery: a new `LoadProjectTestCases(dir)` function finds directories containing `main.sl`
  - Project test cases:
    - Basic package import
    - Multi-file directory package
    - Transitive dependencies
    - Cross-package struct usage
    - Cross-package class method dispatch
    - Transitive type exposure (use returned types without importing origin)
    - Import aliasing
    - Two-file root package without `packages/` directory
  - Package initialization order
  - Circular dependency errors
  - Missing/empty package errors

# Code Examples

## Example 1: Basic Package Import

The simplest multi-package program. A `math` package with one file, imported by main.

```
project/
  main.sl
  packages/
    math/
      math.sl
```

```slang
// packages/math/math.sl
add = (a: s64, b: s64) -> s64 {
    return a + b
}

square = (x: s64) -> s64 {
    return x * x
}
```

```slang
// main.sl
math = import("math")

main = () {
    val result = math.add(3, 4)
    print(result)  // prints: 7

    print(math.square(5))  // prints: 25
}
```

## Example 2: Exporting Structs

Structs and their fields are accessible to importers.

```
project/
  main.sl
  packages/
    geometry/
      geometry.sl
```

```slang
// packages/geometry/geometry.sl
Point = struct {
    val x: s64
    val y: s64
}

distance = (p1: &Point, p2: &Point) -> s64 {
    val dx = p2.x - p1.x
    val dy = p2.y - p1.y
    return dx + dy
}
```

```slang
// main.sl
geo = import("geometry")

main = () {
    val a = geo.Point{ 0, 0 }
    val b = geo.Point{ 3, 4 }
    print(geo.distance(a, b))  // prints: 7
}
```

## Example 3: Directory Package

Multiple files forming a single package. All declarations are shared across files and visible to importers.

```
project/
  main.sl
  packages/
    utils/
      format.sl
      convert.sl
```

```slang
// packages/utils/format.sl
format_s64 = (n: s64) -> s64 {
    return pad(n, 8)
}

pad = (n: s64, width: s64) -> s64 {
    // ... padding logic
    return n
}
```

```slang
// packages/utils/convert.sl
to_hex = (n: s64) -> s64 {
    val raw = n % 16
    return pad(raw, 8)  // OK: 'pad' is in same package (format.sl)
}
```

```slang
// main.sl
utils = import("utils")

main = () {
    print(utils.format_s64(42))
    print(utils.to_hex(255))
    print(utils.pad(7, 8))
}
```

## Example 4: Multiple Imports

A program importing from several packages.

```slang
// main.sl
math = import("math")
strings = import("strings")
config = import("config")

main = () {
    val greeting = strings.concat("Hello, ", config.app_name)
    print(greeting)

    val result = math.add(config.default_x, config.default_y)
    print(result)
}
```

## Example 5: Import Aliasing

The left-hand side of the import assignment is the local name. Use different names to avoid conflicts.

```
project/
  main.sl
  packages/
    math/
      integers/
        integers.sl
      big/
        big.sl
```

```slang
// main.sl
int_math = import("math/integers")
big_math = import("math/big")

main = () {
    val a = int_math.add(1, 2)
    val b = big_math.add(1000000, 2000000)
    print(a)  // prints: 3
    print(b)  // prints: 3000000
}
```

## Example 6: Transitive Dependencies

Package A imports B, B imports C. A does not need to import C unless it uses C's types directly.

```
project/
  main.sl
  packages/
    logger/
      logger.sl
    validator/
      validator.sl
```

```slang
// packages/logger/logger.sl
log = (msg: string) {
    print(msg)
}
```

```slang
// packages/validator/validator.sl
logger = import("logger")

validate = (value: s64) -> bool {
    if value < 0 {
        logger.log("validation failed: negative value")
        return false
    }
    return true
}
```

```slang
// main.sl
validator = import("validator")

main = () {
    val ok = validator.validate(-5)
    if !ok {
        exit(1)
    }
}
```

## Example 7: Circular Dependency Error

The compiler rejects circular imports with a clear error message.

```
project/
  main.sl
  packages/
    a/
      a.sl
    b/
      b.sl
```

```slang
// packages/a/a.sl
b = import("b")
foo = () -> s64 { return b.bar() }
```

```slang
// packages/b/b.sl
a = import("a")
bar = () -> s64 { return a.foo() }
```

```
// @test: expect_error=true
// @test: error_stage=module
// @test: error_contains=circular dependency: a -> b -> a
```

## Example 8: Package Initialization Order

Top-level variable initializers run at runtime, before `main()`, in dependency order.

```
project/
  main.sl
  packages/
    config/
      config.sl
    db/
      db.sl
```

```slang
// packages/config/config.sl
val db_host = "localhost"
val db_port: s64 = 5432
```

```slang
// packages/db/db.sl
config = import("config")

// This runs at init time, after config package initializes
val connection = connect(config.db_host, config.db_port)

connect = (host: string, port: s64) -> s64 {
    // ... returns connection handle
    return 1
}
```

```slang
// main.sl
db = import("db")

// Initialization order:
// 1. config package (no dependencies)
// 2. db package (depends on config)
// 3. main's top-level declarations
// 4. main() is called

main = () {
    print(db.connection)  // connection already initialized
}
```

## Example 9: Cross-Package Class Usage

Classes defined in one package can be used by importers. Methods are resolved through the type's origin package.

```
project/
  main.sl
  packages/
    account/
      account.sl
```

```slang
// packages/account/account.sl
Account = class {
    var balance: s64

    get_balance = (self: &Account) -> s64 {
        return self.balance
    }

    deposit = (self: &&Account, amount: s64) {
        self.balance = self.balance + amount
    }
}
```

```slang
// main.sl
acct = import("account")

main = () {
    val a = acct.Account{ 1000 }
    print(a.get_balance())  // prints: 1000

    a.deposit(500)
    print(a.get_balance())  // prints: 1500
}
```

Note: `a.get_balance()` resolves the method through `a`'s type (`account.Account`), not through the `acct` import alias. The analyzer looks up methods in the `account` package's class definition.

## Example 10: Same-Package Forward References

Files within the same package can reference each other's declarations without imports.

```
project/
  main.sl
  packages/
    game/
      player.sl
      world.sl
```

```slang
// packages/game/player.sl
Player = struct {
    val name: string
    var x: s64
    var y: s64
}

move_player = (player: &&Player, world: &World, dx: s64, dy: s64) {
    // Can reference World from world.sl -- same package, no import needed
    val new_x = player.x + dx
    val new_y = player.y + dy
    if new_x >= 0 && new_x < world.width && new_y >= 0 && new_y < world.height {
        player.x = new_x
        player.y = new_y
    }
}
```

```slang
// packages/game/world.sl
World = struct {
    val width: s64
    val height: s64
}

create_world = (w: s64, h: s64) -> World {
    return World{ w, h }
}
```

```slang
// main.sl
game = import("game")

main = () {
    val world = game.create_world(100, 100)
    var player = game.Player{ "Alice", 50, 50 }
    game.move_player(player, world, 10, -5)
    print(player.x)  // prints: 60
    print(player.y)  // prints: 45
}
```

# Implementation Order

1. [ ] **Lexer** - Add `import` token
2. [ ] **AST** - Add `ImportDecl`, `TopLevelVarDecl`, add `Imports` field to `Program`, add `File` to `Position`
3. [ ] **Parser** - Parse import declarations, qualified type names (dotted strings in `parseTypeName`), qualified struct literals (`PackageAlias` on `StructLiteral`), top-level `val`/`var` as `TopLevelVarDecl`
4. [ ] **SlPackageResolver** - Path resolution within `packages/` directory
5. [ ] **SlPackageCompiler (Phase 1)** - `PackageAST`/`FileAST`, recursive parse-and-discover, cycle detection, topological sort; accepts entry file + explicit root file list; returns `map[string]*PackageAST`
6. [ ] **SlPackageCompiler (Phase 2)** - Consume `PackageAST` values, `AnalyzePackage` with registration + type-checking phases, create `SlPackageNamespace` bindings, populate `SlPackage.TypedAST` and `SlPackage.Exports`; validate `main` is in entry file only
7. [ ] **IR Generator** - Name mangling (`SetPackagePath`), cross-package calls, combined `*ir.Program`, `PackagePath` on semantic types
8. [ ] **ARM64 Backend** - `_sl_` prefix for internal labels, `InitOrder`-driven `_start`, global variable `.data` slots, `main_.main` lookup
9. [ ] **Package Initialization** - Init function generation, global heap allocation in init, ordering via `InitOrder`
10. [ ] **E2E Tests** - All tests through `SlPackageCompiler`; project tests in `_examples/slang/projects/`

# Files Modified

| File | Changes | Status |
|------|---------|--------|
| `compiler/lexer/lexer.go` | Add `TokenTypeImport` | Planned |
| `compiler/ast/ast.go` | Add `ImportDecl`, `TopLevelVarDecl`, add `Imports` to `Program`, add `File` to `Position`, `PackageAlias` on `StructLiteral` | Planned |
| `compiler/parser/parser.go` | Parse `import`, qualified type annotations (dotted strings), qualified struct literals, top-level `val`/`var` | Planned |
| `compiler/slpackage/resolver.go` | New: `SlPackageResolver` -- path resolution within `packages/` | Planned |
| `compiler/slpackage/compiler.go` | New: `SlPackageCompiler`, `SlPackage`, `Export`, `PackageAST`, `FileAST` -- compilation orchestration | Planned |
| `compiler/semantic/analyzer.go` | `SlPackageNamespace`, `AnalyzePackage`, cross-package type checking, method dispatch, `main` validation | Planned |
| `compiler/semantic/types.go` | Add `PackagePath` to `StructType`, `ClassType`, `ObjectType`; update `Equals()` and `String()` | Planned |
| `compiler/ir/generator.go` | `SetPackagePath`, name mangling, cross-package references, combined `*ir.Program` | Planned |
| `compiler/ir/program.go` | Add `InitOrder` field, update `Main()` and `Validate()` for `main_.main` | Planned |
| `compiler/ir/backend/arm64/backend.go` | `_sl_` prefix for internal labels, `InitOrder`-driven `_start`, global `.data` slots | Planned |
| `cmd/sl/main.go` | Pipeline delegates to `SlPackageCompiler`, root file discovery | Planned |
| `test/sl/e2e_test.go` | All tests through `SlPackageCompiler`, project test discovery | Planned |
| `test/testutil/expectations.go` | `LoadProjectTestCases` for directory-based tests | Planned |

# Risks and Limitations

1. **Compilation Speed**: Parsing all transitive dependencies before compiling adds overhead. Mitigated by compiling each package only once.

2. **Name Mangling Complexity**: Mangled names must be deterministic and avoid collisions. Long package paths could produce very long symbol names in assembly output.

3. **Error Reporting Across Packages**: Type errors involving imported types need to show the package path for clarity. Error messages must include the source package.

4. **No Incremental Compilation**: Initially, all packages are recompiled every time. Incremental compilation (only recompiling changed packages) is a future optimization.

5. **Standard Library Bootstrap** (deferred): When standard library support is added, the stdlib will need to be compiled and available before user code. The stdlib will ship alongside the compiler binary.

6. **Directory Package Conflicts**: Two files in the same directory defining the same name is a compile error. Clear error messages are needed.

7. **Single Assembly Output**: All packages are combined into one `.s` file. For very large programs this could be slow to assemble. Per-package object files could be a future optimization.

8. **Initialization Side Effects**: Top-level variable initializers run before `main()`. If an initializer panics or calls `exit()`, the program terminates before reaching `main()`. The runtime generates stack traces for any runtime failures that occur during initialization, including the package name and file where the failure originated, so that debugging init-time crashes is tractable.

9. **Directory Overhead for Simple Modules**: Every importable unit must be a directory within `packages/`, even for a single-file module. This means `packages/math/math.sl` instead of just `math.sl`. The trade-off is simplicity and unambiguous resolution. The `packages/` directory adds one level of nesting but provides clear project structure in return.

10. **Breaking Change: `import` Reserved Keyword**: `import` becomes a reserved keyword and can no longer be used as an identifier (variable, function, or parameter name). Existing code using `import` as an identifier will produce a parse error. Since Slang is pre-1.0, this is an acceptable migration cost.

# Future Enhancements

These are explicitly out of scope but may be added later:

1. **Visibility Modifiers** (separate SEP)
   - `private` keyword for file-level and member-level access control
   - Possibly `internal` for package-level visibility

2. **Selective Imports**
   ```slang
   // Import specific symbols into local scope
   { add, subtract } = import("math")
   val result = add(1, 2)  // no prefix needed
   ```

3. **Re-exports**
   ```slang
   // Forward a package's public API
   // (syntax TBD)
   ```

4. **Conditional Compilation**
   ```slang
   // Platform-specific imports
   os = import(when { platform == "macos" -> "os_macos", else -> "os_linux" })
   ```

5. **Package Manager**
   - Version resolution and dependency management
   - Central package registry
   - Lock files for reproducible builds

6. **Incremental Compilation**
   - Cache compiled package IR
   - Only recompile changed packages and their dependents

7. **Standard Library**
   ```slang
   // Named imports for standard library packages
   math = import("std/math")
   io = import("std/io")
   ```
   - Ships alongside the compiler binary
   - Resolved via named paths (non-relative)

8. **Separate Compilation Units**
   - Compile each package to its own object file
   - Link with system linker for faster builds
