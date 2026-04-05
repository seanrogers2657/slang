package slpackage

import (
	"fmt"
	"os"

	"path/filepath"
	"strings"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/ir"
	"github.com/seanrogers2657/slang/compiler/lexer"
	"github.com/seanrogers2657/slang/compiler/parser"
	"github.com/seanrogers2657/slang/compiler/semantic"
	"github.com/seanrogers2657/slang/errors"
)

// Package represents a single compilation unit (one directory of .sl files).
type Package struct {
	Path string // import path ("main" for root, "math" for packages/math/)
	Dir  string // absolute directory path on disk
}

// PackageCompiler orchestrates compilation across all packages.
type PackageCompiler struct {
	RootDir       string              // project root directory
	EntryFile     string              // the specific file passed to sl build/run
	RootFiles     []string            // all .sl files for root package
	Resolver      *PackageResolver    // translates import paths to directories
	Packages      map[string]*Package        // all discovered packages
	AnalysisOrder []string                   // topological order of package paths
	Warnings      []*errors.CompilerError   // non-fatal warnings

	// internal state for discovery
	importGraph map[string][]string // package path -> imported package paths
}

// NewCompiler creates a PackageCompiler for the given project.
func NewCompiler(rootDir, entryFile string, rootFiles []string) *PackageCompiler {
	return &PackageCompiler{
		RootDir:     rootDir,
		EntryFile:   entryFile,
		RootFiles:   rootFiles,
		Resolver:    NewResolver(rootDir),
		Packages:    make(map[string]*Package),
		importGraph: make(map[string][]string),
	}
}

// DiscoverAndParse executes Phase 1: discovery, lexing, parsing, cycle detection, and topological sort.
// Returns per-package file lists keyed by package path.
// For a single-file program with no imports, returns one entry for "main" with one FileAST.
func (c *PackageCompiler) DiscoverAndParse() (map[string][]*ast.FileAST, []*errors.CompilerError) {
	var allErrors []*errors.CompilerError
	pkgFiles := make(map[string][]*ast.FileAST)

	// Parse root package
	c.Packages["main"] = &Package{Path: "main", Dir: c.RootDir}
	rootFileASTs, parseErrs := c.parseFiles(c.RootFiles)
	allErrors = append(allErrors, parseErrs...)
	pkgFiles["main"] = rootFileASTs

	// Extract imports from root package and discover dependencies
	if len(parseErrs) == 0 {
		discoverErrs := c.discoverImports("main", rootFileASTs, pkgFiles)
		allErrors = append(allErrors, discoverErrs...)
	}

	// Warn about loose .sl files directly in packages/
	c.checkLooseFiles()

	// Cycle detection
	if cycleErr := c.detectCycles(); cycleErr != nil {
		allErrors = append(allErrors, cycleErr)
	}

	// Topological sort
	if len(allErrors) == 0 {
		c.AnalysisOrder = c.topologicalSort()
	}

	if len(allErrors) > 0 {
		return nil, allErrors
	}

	return pkgFiles, nil
}

// parseFiles reads, lexes, and parses a list of .sl files.
func (c *PackageCompiler) parseFiles(filePaths []string) ([]*ast.FileAST, []*errors.CompilerError) {
	var fileASTs []*ast.FileAST
	var allErrors []*errors.CompilerError

	for _, filePath := range filePaths {
		source, err := os.ReadFile(filePath)
		if err != nil {
			allErrors = append(allErrors, errors.NewError(
				fmt.Sprintf("cannot read file: %s", err),
				filePath, errors.Position{}, "module",
			))
			continue
		}

		// Lex
		l := lexer.NewLexerWithFilename(source, filePath)
		l.Parse()
		if len(l.Errors) > 0 {
			allErrors = append(allErrors, l.Errors...)
			continue // don't parse if lexer failed
		}

		// Parse
		p := parser.NewParser(l.Tokens)
		program := p.Parse()
		if len(p.Errors) > 0 {
			for _, e := range p.Errors {
				allErrors = append(allErrors, e)
			}
			continue // don't extract imports if parser failed
		}

		fileASTs = append(fileASTs, &ast.FileAST{
			Path: filePath,
			AST:  program,
		})
	}

	return fileASTs, allErrors
}

// discoverImports extracts imports from a package's file ASTs and recursively discovers dependencies.
func (c *PackageCompiler) discoverImports(pkgPath string, fileASTs []*ast.FileAST, pkgFiles map[string][]*ast.FileAST) []*errors.CompilerError {
	var allErrors []*errors.CompilerError

	for _, fileAST := range fileASTs {
		for _, imp := range fileAST.AST.Imports {
			// Convert ast.Position to errors.Position for error reporting
			errPos := errors.Position{
				Line:   imp.ImportPos.Line,
				Column: imp.ImportPos.Column,
				Offset: imp.ImportPos.Offset,
			}

			// Resolve the import path
			pkgDir, err := c.Resolver.Resolve(imp.Path)
			if err != nil {
				allErrors = append(allErrors, errors.NewError(
					err.Error(), fileAST.Path, errPos, "module",
				))
				continue
			}

			// Record import edge
			c.importGraph[pkgPath] = append(c.importGraph[pkgPath], imp.Path)

			// Skip if already discovered
			if _, exists := c.Packages[imp.Path]; exists {
				continue
			}

			// Register the new package
			c.Packages[imp.Path] = &Package{Path: imp.Path, Dir: pkgDir}

			// Discover .sl files in the package directory
			files, err := DiscoverSlFiles(pkgDir)
			if err != nil {
				allErrors = append(allErrors, errors.NewError(
					fmt.Sprintf("error reading package %q: %s", imp.Path, err),
					fileAST.Path, errPos, "module",
				))
				continue
			}

			// Parse the package files
			depFileASTs, parseErrs := c.parseFiles(files)
			allErrors = append(allErrors, parseErrs...)
			pkgFiles[imp.Path] = depFileASTs

			// Recursively discover imports from this package
			if len(parseErrs) == 0 {
				depErrs := c.discoverImports(imp.Path, depFileASTs, pkgFiles)
				allErrors = append(allErrors, depErrs...)
			}
		}
	}

	return allErrors
}

// detectCycles checks for circular dependencies using DFS.
// Returns an error describing the cycle if one is found.
func (c *PackageCompiler) detectCycles() *errors.CompilerError {
	const (
		white = 0 // unvisited
		gray  = 1 // in progress
		black = 2 // done
	)

	colors := make(map[string]int)
	parent := make(map[string]string)

	var dfs func(node string) *errors.CompilerError
	dfs = func(node string) *errors.CompilerError {
		colors[node] = gray

		for _, dep := range c.importGraph[node] {
			if colors[dep] == gray {
				// Found a cycle — reconstruct the path
				cycle := []string{dep, node}
				cur := node
				for cur != dep {
					cur = parent[cur]
					cycle = append(cycle, cur)
				}
				// Reverse to get forward order
				for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
					cycle[i], cycle[j] = cycle[j], cycle[i]
				}
				cycleStr := ""
				for i, p := range cycle {
					if i > 0 {
						cycleStr += " -> "
					}
					cycleStr += p
				}
				return errors.NewError(
					fmt.Sprintf("circular dependency detected: %s", cycleStr),
					"", errors.Position{}, "module",
				)
			}
			if colors[dep] == white {
				parent[dep] = node
				if err := dfs(dep); err != nil {
					return err
				}
			}
		}

		colors[node] = black
		return nil
	}

	for pkg := range c.Packages {
		if colors[pkg] == white {
			if err := dfs(pkg); err != nil {
				return err
			}
		}
	}

	return nil
}

// topologicalSort returns package paths in dependency order (dependencies first).
func (c *PackageCompiler) topologicalSort() []string {
	visited := make(map[string]bool)
	var order []string

	var visit func(node string)
	visit = func(node string) {
		if visited[node] {
			return
		}
		visited[node] = true

		for _, dep := range c.importGraph[node] {
			visit(dep)
		}

		order = append(order, node)
	}

	// Visit all packages (deterministic order by visiting "main" last)
	for pkg := range c.Packages {
		if pkg != "main" {
			visit(pkg)
		}
	}
	visit("main")

	return order
}

// Analyze executes Phase 2: semantic analysis for all packages in dependency order.
// Returns typed programs keyed by package path, or errors if analysis fails.
func (c *PackageCompiler) Analyze(pkgFiles map[string][]*ast.FileAST) ([]*errors.CompilerError, map[string]*semantic.TypedProgram) {
	typedPrograms := make(map[string]*semantic.TypedProgram)
	packageNamespaces := make(map[string]*semantic.PackageNamespace)

	for _, path := range c.AnalysisOrder {
		files := pkgFiles[path]

		// Determine filename for the analyzer (use entry file for root, first file for others)
		filename := ""
		if path == "main" && c.EntryFile != "" {
			filename = c.EntryFile
		} else if len(files) > 0 {
			filename = files[0].Path
		}

		// Build deps from already-analyzed packages that this package imports
		var deps map[string]*semantic.PackageNamespace
		if imports := c.importGraph[path]; len(imports) > 0 {
			deps = make(map[string]*semantic.PackageNamespace)
			for _, impPath := range imports {
				if ns, ok := packageNamespaces[impPath]; ok {
					deps[impPath] = ns
				}
			}
		}

		analyzer := semantic.NewAnalyzer(filename)
		isRoot := path == "main"
		errs, typedAST := analyzer.AnalyzePackage(files, path, isRoot, deps)
		if len(errs) > 0 {
			return errs, nil
		}

		typedPrograms[path] = typedAST

		// Extract exports for downstream packages
		exports := semantic.ExtractExports(typedAST)

		// Validate: imported packages must not define 'main'
		if !isRoot {
			if _, hasMain := exports["main"]; hasMain {
				return []*errors.CompilerError{
					errors.NewError(
						fmt.Sprintf("package '%s' must not declare a 'main' function", path),
						filename, errors.Position{}, "module",
					),
				}, nil
			}
		}

		packageNamespaces[path] = &semantic.PackageNamespace{
			Path:    path,
			Exports: exports,
		}
	}

	return nil, typedPrograms
}

// GenerateIR executes Phase 3: generates IR for all packages in dependency order,
// mangles cross-package names, and combines into a single *ir.Program.
func (c *PackageCompiler) GenerateIR(typedPrograms map[string]*semantic.TypedProgram) (*ir.Program, error) {
	var combined *ir.Program

	// First pass: collect globals (mutable top-level vars from all packages)
	globalVars := make(map[string]bool)
	for _, pkgPath := range c.AnalysisOrder {
		typedAST := typedPrograms[pkgPath]
		if typedAST == nil {
			continue
		}
		prefix := ""
		if pkgPath != "main" {
			prefix = ManglePrefix(pkgPath)
		}
		for _, stmt := range typedAST.Statements {
			if varDecl, ok := stmt.(*semantic.TypedVarDeclStmt); ok && varDecl.Mutable {
				globalVars[prefix+varDecl.Name] = true
			}
		}
	}

	// Collect all top-level statements from all packages in dependency order
	var allTopLevelStmts []ir.PrefixedStmt
	for _, pkgPath := range c.AnalysisOrder {
		typedAST := typedPrograms[pkgPath]
		if typedAST == nil {
			continue
		}
		prefix := ""
		if pkgPath != "main" {
			prefix = ManglePrefix(pkgPath)
		}
		for _, stmt := range typedAST.Statements {
			allTopLevelStmts = append(allTopLevelStmts, ir.PrefixedStmt{Stmt: stmt, Prefix: prefix})
		}
	}

	// Second pass: generate IR for each package
	for _, pkgPath := range c.AnalysisOrder {
		typedAST := typedPrograms[pkgPath]
		if typedAST == nil {
			continue
		}

		prefix := ""
		if pkgPath != "main" {
			prefix = ManglePrefix(pkgPath)
			typedAST.Statements = nil // non-root stmts handled via TopLevelStmts
		}

		config := ir.GeneratorConfig{
			PackagePrefix: prefix,
			GlobalVars:    globalVars,
		}
		if pkgPath == "main" {
			config.TopLevelStmts = allTopLevelStmts
			typedAST.Statements = nil // handled via TopLevelStmts
		}

		g := ir.NewGenerator(config)
		prog, err := g.GenerateProgram(typedAST)
		if err != nil {
			return nil, fmt.Errorf("IR generation error for package %s: %w", pkgPath, err)
		}

		// Mangle function names and intra-package call references for non-root packages
		if pkgPath != "main" {
			localFns := make(map[string]bool)
			for _, fn := range prog.Functions {
				localFns[fn.Name] = true
			}
			for _, fn := range prog.Functions {
				fn.Name = prefix + fn.Name
			}
			for _, fn := range prog.Functions {
				for _, block := range fn.Blocks {
					for _, v := range block.Values {
						if v.Op == ir.OpCall && localFns[v.AuxString] {
							v.AuxString = prefix + v.AuxString
						}
					}
				}
			}
		}

		if combined == nil {
			combined = prog
		} else {
			combined.Functions = append(combined.Functions, prog.Functions...)
			combined.Globals = append(combined.Globals, prog.Globals...)
		}
	}

	return combined, nil
}

// checkLooseFiles warns about .sl files directly in packages/ directory.
// Warnings are added to the Warnings field, not to the error list.
func (c *PackageCompiler) checkLooseFiles() {
	packagesDir := c.Resolver.PackagesDir
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return // packages/ doesn't exist, which is fine
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sl") {
			c.Warnings = append(c.Warnings, errors.NewWarning(
				fmt.Sprintf("file '%s' is directly in 'packages/' and is not part of any package; move it into a subdirectory", entry.Name()),
				filepath.Join(packagesDir, entry.Name()),
				errors.Position{}, "module",
			))
		}
	}
}

// ManglePrefix returns the mangled prefix for a package path.
// "math" -> "math__", "utils/helpers" -> "utils__helpers__"
func ManglePrefix(packagePath string) string {
	return strings.ReplaceAll(packagePath, "/", "__") + "__"
}

// MangleName produces a mangled name for a cross-package symbol.
// "math" + "add" -> "math__add"
func MangleName(packagePath string, name string) string {
	return ManglePrefix(packagePath) + name
}
