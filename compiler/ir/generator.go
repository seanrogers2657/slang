package ir

import (
	"fmt"
	"strconv"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

// binaryOpMap maps operator strings to IR opcodes for standard binary operations.
var binaryOpMap = map[string]Op{
	"+":  OpAdd,
	"-":  OpSub,
	"*":  OpMul,
	"/":  OpDiv,
	"%":  OpMod,
	"==": OpEq,
	"!=": OpNe,
	"<":  OpLt,
	"<=": OpLe,
	">":  OpGt,
	">=": OpGe,
}

// ownedVar tracks an owned pointer variable for automatic cleanup
type ownedVar struct {
	name    string
	semType semantic.Type
}

// Generator converts a TypedProgram into IR.
type Generator struct {
	prog *Program

	// Current function being generated
	fn *Function

	// Current block being generated
	block *Block

	// SSA construction
	ssa *SSABuilder

	// Type mapping from semantic types to IR types
	typeCache map[semantic.Type]Type

	// Loop control flow targets
	breakTarget    *Block
	continueTarget *Block

	// Scope tracking for owned pointer cleanup
	// Each scope level contains owned pointers declared at that level
	ownedVarScopes [][]ownedVar

	// Track variables whose ownership has been transferred (moved)
	// These should not be freed during cleanup
	movedVars map[string]bool
}

// NewGenerator creates a new IR generator.
func NewGenerator() *Generator {
	return &Generator{
		prog:      NewProgram(),
		ssa:       NewSSABuilder(),
		typeCache: make(map[semantic.Type]Type),
	}
}

// pushScope creates a new scope for tracking owned pointers.
func (g *Generator) pushScope() {
	g.ownedVarScopes = append(g.ownedVarScopes, nil)
}

// popScope cleans up owned pointers in the current scope and removes it.
func (g *Generator) popScope() {
	if len(g.ownedVarScopes) == 0 {
		return
	}
	// Emit cleanup for all owned pointers in this scope
	g.emitScopeCleanup()
	// Remove the scope
	g.ownedVarScopes = g.ownedVarScopes[:len(g.ownedVarScopes)-1]
}

// trackOwnedVar registers an owned pointer variable for cleanup when scope exits.
func (g *Generator) trackOwnedVar(name string, semType semantic.Type) {
	if len(g.ownedVarScopes) == 0 {
		return
	}
	// Only track if it's actually an owned pointer type
	if elemType, _ := g.getOwnedPointerInfo(semType); elemType != nil {
		lastIdx := len(g.ownedVarScopes) - 1
		g.ownedVarScopes[lastIdx] = append(g.ownedVarScopes[lastIdx], ownedVar{name, semType})
	}
}

// emitScopeCleanup emits free operations for all owned pointers in the current scope.
func (g *Generator) emitScopeCleanup() {
	if len(g.ownedVarScopes) == 0 || g.block == nil {
		return
	}
	lastIdx := len(g.ownedVarScopes) - 1
	for _, ov := range g.ownedVarScopes[lastIdx] {
		// Skip variables that have been moved (ownership transferred)
		if g.movedVars[ov.name] {
			continue
		}
		g.emitFreeIfOwned(ov.name, ov.semType)
	}
}

// emitAllScopesCleanup emits cleanup for all scopes (for function return).
func (g *Generator) emitAllScopesCleanup(excludeVar string) {
	if g.block == nil {
		return
	}
	// Free owned pointers from all scopes, innermost first
	for i := len(g.ownedVarScopes) - 1; i >= 0; i-- {
		for _, ov := range g.ownedVarScopes[i] {
			// Skip variables that have been moved or are being returned
			if ov.name == excludeVar || g.movedVars[ov.name] {
				continue
			}
			g.emitFreeIfOwned(ov.name, ov.semType)
		}
	}
}

// markMoved marks a variable as moved (ownership transferred).
// Moved variables will not be freed during cleanup.
func (g *Generator) markMoved(name string) {
	if g.movedVars == nil {
		g.movedVars = make(map[string]bool)
	}
	g.movedVars[name] = true
}

// Generate converts a TypedProgram to IR.
func Generate(typed *semantic.TypedProgram) (*Program, error) {
	g := NewGenerator()
	return g.GenerateProgram(typed)
}

// GenerateProgram generates IR for an entire program.
func (g *Generator) GenerateProgram(typed *semantic.TypedProgram) (*Program, error) {
	// First pass: register all struct types
	for _, decl := range typed.Declarations {
		if sd, ok := decl.(*semantic.TypedStructDecl); ok {
			g.registerStruct(sd)
		}
	}

	// Second pass: generate all functions
	for _, decl := range typed.Declarations {
		switch d := decl.(type) {
		case *semantic.TypedFunctionDecl:
			if err := g.generateFunction(d); err != nil {
				return nil, err
			}
		case *semantic.TypedClassDecl:
			if err := g.generateClass(d); err != nil {
				return nil, err
			}
		case *semantic.TypedObjectDecl:
			if err := g.generateObject(d); err != nil {
				return nil, err
			}
		}
	}

	return g.prog, nil
}

// registerStruct adds a struct type to the program.
func (g *Generator) registerStruct(sd *semantic.TypedStructDecl) {
	st := g.convertStructType(&sd.StructType)
	g.prog.AddStruct(st)
}

// generateFunction generates IR for a function declaration.
func (g *Generator) generateFunction(fd *semantic.TypedFunctionDecl) error {
	// Convert return type
	retType := g.convertType(fd.ReturnType)

	// Create function
	g.fn = g.prog.NewFunction(fd.Name, retType)

	// Reset SSA state for this function
	g.ssa.Reset()
	g.ssa.SetFunction(g.fn)

	// Reset owned pointer scope tracking for this function
	g.ownedVarScopes = nil
	g.movedVars = make(map[string]bool)
	g.pushScope()

	// Create entry block
	g.block = g.fn.NewBlock(BlockPlain)

	// Generate parameters
	for _, param := range fd.Parameters {
		paramType := g.convertType(param.Type)
		paramVal := g.fn.NewParam(paramType)

		// Record parameter as initial definition of the variable
		g.writeVariable(param.Name, g.block, paramVal)

		// Track owned pointer parameters for cleanup
		g.trackOwnedVar(param.Name, param.Type)
	}

	// Generate function body
	if fd.Body != nil {
		if err := g.generateBlock(fd.Body); err != nil {
			return err
		}
	}

	// Seal all blocks (we're done adding predecessors)
	for _, b := range g.fn.Blocks {
		g.sealBlock(b)
	}

	// If the function is void and doesn't end with a return, add one
	if g.fn.IsVoid() && g.block != nil && g.block.Kind == BlockPlain {
		// Clean up all owned pointers before implicit return
		g.emitAllScopesCleanup("")
		g.addReturn(nil)
	}

	return nil
}

// generateClass generates IR for a class declaration (methods become functions).
func (g *Generator) generateClass(cd *semantic.TypedClassDecl) error {
	// Register the class type as a struct
	st := g.convertClassType(&cd.ClassType)
	g.prog.AddStruct(st)

	// Generate each method as a function with mangled name
	for _, method := range cd.Methods {
		if err := g.generateMethod(cd.Name, method); err != nil {
			return err
		}
	}

	return nil
}

// generateObject generates IR for an object declaration (methods become functions).
func (g *Generator) generateObject(od *semantic.TypedObjectDecl) error {
	// Generate each method as a function with mangled name
	for _, method := range od.Methods {
		if err := g.generateMethod(od.Name, method); err != nil {
			return err
		}
	}

	return nil
}

// generateMethod generates IR for a method declaration.
func (g *Generator) generateMethod(className string, md *semantic.TypedMethodDecl) error {
	// Mangle name: ClassName_methodName_paramCount (for overloading support)
	mangledName := fmt.Sprintf("%s_%s_%d", className, md.Name, len(md.Parameters))

	// Convert return type
	retType := g.convertType(md.ReturnType)

	// Create function
	g.fn = g.prog.NewFunction(mangledName, retType)

	// Reset SSA state
	g.ssa.Reset()
	g.ssa.SetFunction(g.fn)

	// Create entry block
	g.block = g.fn.NewBlock(BlockPlain)

	// Generate parameters (including self for instance methods)
	for _, param := range md.Parameters {
		paramType := g.convertType(param.Type)
		paramVal := g.fn.NewParam(paramType)
		g.writeVariable(param.Name, g.block, paramVal)
	}

	// Generate method body
	if md.Body != nil {
		if err := g.generateBlock(md.Body); err != nil {
			return err
		}
	}

	// Seal all blocks
	for _, b := range g.fn.Blocks {
		g.sealBlock(b)
	}

	// Add void return if needed
	if g.fn.IsVoid() && g.block != nil && g.block.Kind == BlockPlain {
		g.addReturn(nil)
	}

	return nil
}

// generateBlock generates IR for a block statement.
func (g *Generator) generateBlock(bs *semantic.TypedBlockStmt) error {
	for _, stmt := range bs.Statements {
		if err := g.generateStatement(stmt); err != nil {
			return err
		}
		// Stop if we've terminated the block
		if g.block == nil || g.block.Kind != BlockPlain {
			break
		}
	}
	return nil
}

// generateStatement generates IR for a statement.
func (g *Generator) generateStatement(stmt semantic.TypedStatement) error {
	switch s := stmt.(type) {
	case *semantic.TypedExprStmt:
		_, err := g.generateExpr(s.Expr)
		return err

	case *semantic.TypedVarDeclStmt:
		return g.generateVarDecl(s)

	case *semantic.TypedAssignStmt:
		return g.generateAssign(s)

	case *semantic.TypedFieldAssignStmt:
		return g.generateFieldAssign(s)

	case *semantic.TypedIndexAssignStmt:
		return g.generateIndexAssign(s)

	case *semantic.TypedReturnStmt:
		return g.generateReturn(s)

	case *semantic.TypedIfStmt:
		// If it has a ResultType, treat as expression (for nested if expressions)
		if s.ResultType != nil {
			_, err := g.generateIfExpr(s)
			return err
		}
		return g.generateIf(s)

	case *semantic.TypedWhileStmt:
		return g.generateWhile(s)

	case *semantic.TypedForStmt:
		return g.generateFor(s)

	case *semantic.TypedBreakStmt:
		return g.generateBreak(s)

	case *semantic.TypedContinueStmt:
		return g.generateContinue(s)

	case *semantic.TypedBlockStmt:
		return g.generateBlock(s)

	case *semantic.TypedWhenExpr:
		_, err := g.generateWhen(s)
		return err

	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// generateVarDecl generates IR for a variable declaration.
func (g *Generator) generateVarDecl(vd *semantic.TypedVarDeclStmt) error {
	declType := g.convertType(vd.DeclaredType)

	// Handle null literal specially
	if isNullLiteral(vd.Initializer) {
		g.writeVariable(vd.Name, g.block, g.block.NewValue(OpWrapNull, declType))
		g.trackOwnedVar(vd.Name, vd.DeclaredType)
		return nil
	}

	// Generate initializer and wrap if needed
	val, err := g.generateExpr(vd.Initializer)
	if err != nil {
		return err
	}

	g.writeVariable(vd.Name, g.block, g.wrapIfNeeded(val, declType))
	g.trackOwnedVar(vd.Name, vd.DeclaredType)
	return nil
}

// generateAssign generates IR for a variable assignment.
func (g *Generator) generateAssign(as *semantic.TypedAssignStmt) error {
	varType := g.convertType(as.VarType)

	// Handle null literal specially - free old value first
	if isNullLiteral(as.Value) {
		// Free the old value before setting to null
		g.emitFreeIfOwned(as.Name, as.VarType)
		g.writeVariable(as.Name, g.block, g.block.NewValue(OpWrapNull, varType))
		return nil
	}

	// Generate value and wrap if needed
	val, err := g.generateExpr(as.Value)
	if err != nil {
		return err
	}

	g.writeVariable(as.Name, g.block, g.wrapIfNeeded(val, varType))
	return nil
}

// generateFieldAssign generates IR for a field assignment.
func (g *Generator) generateFieldAssign(fa *semantic.TypedFieldAssignStmt) error {
	// Generate object pointer
	obj, err := g.generateExpr(fa.Object)
	if err != nil {
		return err
	}

	// Generate value to store
	val, err := g.generateExpr(fa.Value)
	if err != nil {
		return err
	}

	// Get field offset
	offset := g.getFieldOffset(fa.Object.GetType(), fa.Field)

	// Create field pointer
	fieldPtr := g.block.NewValue(OpFieldPtr, &PtrType{Elem: val.Type})
	fieldPtr.AddArg(obj)
	fieldPtr.AuxInt = int64(offset)

	// Store value
	store := g.block.NewValue(OpStore, nil)
	store.AddArg(fieldPtr)
	store.AddArg(val)

	return nil
}

// generateIndexAssign generates IR for an array index assignment.
func (g *Generator) generateIndexAssign(ia *semantic.TypedIndexAssignStmt) error {
	// Generate array
	arr, err := g.generateExpr(ia.Array)
	if err != nil {
		return err
	}

	// Generate index
	idx, err := g.generateExpr(ia.Index)
	if err != nil {
		return err
	}

	// Generate value
	val, err := g.generateExpr(ia.Value)
	if err != nil {
		return err
	}

	// Create index pointer
	elemPtr := g.block.NewValue(OpIndexPtr, &PtrType{Elem: val.Type})
	elemPtr.AddArg(arr)
	elemPtr.AddArg(idx)

	// Store value
	store := g.block.NewValue(OpStore, nil)
	store.AddArg(elemPtr)
	store.AddArg(val)

	return nil
}

// generateReturn generates IR for a return statement.
func (g *Generator) generateReturn(rs *semantic.TypedReturnStmt) error {
	var retVal *Value
	var excludeVar string // Variable to exclude from cleanup if returning owned pointer

	if rs.Value != nil {
		retType := g.fn.ReturnType

		// Handle null literal specially
		if isNullLiteral(rs.Value) {
			retVal = g.block.NewValue(OpWrapNull, retType)
		} else {
			// Check if we're returning an identifier that's an owned pointer
			// If so, we transfer ownership rather than freeing it
			if ident, ok := rs.Value.(*semantic.TypedIdentifierExpr); ok {
				if elemType, _ := g.getOwnedPointerInfo(ident.Type); elemType != nil {
					excludeVar = ident.Name
				}
			}

			var err error
			retVal, err = g.generateExpr(rs.Value)
			if err != nil {
				return err
			}
			retVal = g.wrapIfNeeded(retVal, retType)
		}
	}

	// Clean up all owned pointers before returning (except the one being returned)
	g.emitAllScopesCleanup(excludeVar)

	g.addReturn(retVal)
	return nil
}

// addReturn adds a return terminator to the current block.
func (g *Generator) addReturn(val *Value) {
	ret := g.block.NewValue(OpReturn, nil)
	if val != nil {
		ret.AddArg(val)
	}
	g.block.Kind = BlockReturn
	g.block = nil // Block is terminated
}

// generateIf generates IR for an if statement.
func (g *Generator) generateIf(is *semantic.TypedIfStmt) error {
	// Generate condition
	cond, err := g.generateExpr(is.Condition)
	if err != nil {
		return err
	}

	// Create blocks
	thenBlock := g.fn.NewBlock(BlockPlain)
	var elseBlock *Block
	if is.ElseBranch != nil {
		elseBlock = g.fn.NewBlock(BlockPlain)
	}
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Set up conditional branch
	g.block.Kind = BlockIf
	g.block.Control = cond
	g.block.AddSucc(thenBlock)
	if elseBlock != nil {
		g.block.AddSucc(elseBlock)
	} else {
		g.block.AddSucc(mergeBlock)
	}

	// Seal thenBlock - its only predecessor is the current (condition) block
	g.sealBlock(thenBlock)

	// Generate then branch
	g.block = thenBlock
	if err := g.generateBlock(is.ThenBranch); err != nil {
		return err
	}
	// Jump to merge if not terminated
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(mergeBlock)
	}

	// Generate else branch if present
	if is.ElseBranch != nil {
		// Seal elseBlock - its only predecessor is the condition block
		g.sealBlock(elseBlock)

		g.block = elseBlock
		if err := g.generateStatement(is.ElseBranch); err != nil {
			return err
		}
		// Jump to merge if not terminated
		if g.block != nil && g.block.Kind == BlockPlain {
			g.block.AddSucc(mergeBlock)
		}
	}

	// Seal mergeBlock - all predecessors are now known
	g.sealBlock(mergeBlock)

	// Continue in merge block
	g.block = mergeBlock

	return nil
}

// generateWhile generates IR for a while loop.
func (g *Generator) generateWhile(ws *semantic.TypedWhileStmt) error {
	// Create blocks
	headerBlock := g.fn.NewBlock(BlockPlain)
	bodyBlock := g.fn.NewBlock(BlockPlain)
	exitBlock := g.fn.NewBlock(BlockPlain)

	// Save loop context for break/continue
	prevBreakTarget := g.breakTarget
	prevContinueTarget := g.continueTarget
	g.breakTarget = exitBlock
	g.continueTarget = headerBlock

	// Jump to header
	g.block.AddSucc(headerBlock)
	g.block = headerBlock

	// Generate condition
	cond, err := g.generateExpr(ws.Condition)
	if err != nil {
		return err
	}

	// Conditional branch
	g.block.Kind = BlockIf
	g.block.Control = cond
	g.block.AddSucc(bodyBlock)
	g.block.AddSucc(exitBlock)

	// Seal bodyBlock now - its only predecessor is headerBlock
	g.sealBlock(bodyBlock)

	// Generate body
	g.block = bodyBlock
	if err := g.generateBlock(ws.Body); err != nil {
		return err
	}
	// Jump back to header if not terminated
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(headerBlock)
	}

	// Seal headerBlock now - all predecessors are known (entry + body end)
	g.sealBlock(headerBlock)

	// Seal exitBlock - its only predecessor is headerBlock
	g.sealBlock(exitBlock)

	// Restore loop context
	g.breakTarget = prevBreakTarget
	g.continueTarget = prevContinueTarget

	// Continue in exit block
	g.block = exitBlock

	return nil
}


// generateFor generates IR for a for loop.
func (g *Generator) generateFor(fs *semantic.TypedForStmt) error {
	// Generate init (if present)
	if fs.Init != nil {
		if err := g.generateStatement(fs.Init); err != nil {
			return err
		}
	}

	// Create blocks
	headerBlock := g.fn.NewBlock(BlockPlain)
	bodyBlock := g.fn.NewBlock(BlockPlain)
	updateBlock := g.fn.NewBlock(BlockPlain)
	exitBlock := g.fn.NewBlock(BlockPlain)

	// Save loop context for break/continue
	prevBreakTarget := g.breakTarget
	prevContinueTarget := g.continueTarget
	g.breakTarget = exitBlock
	g.continueTarget = updateBlock

	// Jump to header
	g.block.AddSucc(headerBlock)
	g.block = headerBlock

	// Generate condition (if present)
	if fs.Condition != nil {
		cond, err := g.generateExpr(fs.Condition)
		if err != nil {
			return err
		}

		// Conditional branch
		g.block.Kind = BlockIf
		g.block.Control = cond
		g.block.AddSucc(bodyBlock)
		g.block.AddSucc(exitBlock)
	} else {
		// No condition means always true
		g.block.AddSucc(bodyBlock)
	}

	// Seal bodyBlock now - its only predecessor is headerBlock
	g.sealBlock(bodyBlock)

	// Generate body
	g.block = bodyBlock
	if err := g.generateBlock(fs.Body); err != nil {
		return err
	}
	// Jump to update if not terminated
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(updateBlock)
	}

	// Seal updateBlock now - after body is generated (continue statements may add predecessors)
	g.sealBlock(updateBlock)

	// Generate update (in update block)
	g.block = updateBlock
	if fs.Update != nil {
		if err := g.generateStatement(fs.Update); err != nil {
			return err
		}
	}
	// Jump back to header
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(headerBlock)
	}

	// Seal headerBlock now - all its predecessors are known (entry + updateBlock)
	g.sealBlock(headerBlock)

	// Seal exitBlock - after body is generated (break statements may add predecessors)
	g.sealBlock(exitBlock)

	// Restore loop context
	g.breakTarget = prevBreakTarget
	g.continueTarget = prevContinueTarget

	// Continue in exit block
	g.block = exitBlock

	return nil
}

// generateBreak generates IR for a break statement.
func (g *Generator) generateBreak(_ *semantic.TypedBreakStmt) error {
	if g.breakTarget == nil {
		return fmt.Errorf("break outside of loop")
	}
	g.block.AddSucc(g.breakTarget)
	g.block = nil // Block is terminated
	return nil
}

// generateContinue generates IR for a continue statement.
func (g *Generator) generateContinue(_ *semantic.TypedContinueStmt) error {
	if g.continueTarget == nil {
		return fmt.Errorf("continue outside of loop")
	}
	g.block.AddSucc(g.continueTarget)
	g.block = nil // Block is terminated
	return nil
}

// generateWhen generates IR for a when expression.
func (g *Generator) generateWhen(we *semantic.TypedWhenExpr) (*Value, error) {
	resultType := g.convertType(we.ResultType)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Track phi arguments if this is an expression (has result type)
	var phiArgs []*PhiArg

	for i, c := range we.Cases {
		if c.IsElse {
			// Else case: generate body and jump to merge
			if err := g.generateStatement(c.Body); err != nil {
				return nil, err
			}

			// If expression, get result value
			if resultType != nil && g.block != nil {
				val := g.getLastValue()
				if val != nil {
					phiArgs = append(phiArgs, &PhiArg{From: g.block, Value: val})
				}
			}

			// Jump to merge
			if g.block != nil && g.block.Kind == BlockPlain {
				g.block.AddSucc(mergeBlock)
			}
		} else {
			// Regular case: condition -> then/next
			cond, err := g.generateExpr(c.Condition)
			if err != nil {
				return nil, err
			}

			thenBlock := g.fn.NewBlock(BlockPlain)
			var nextBlock *Block
			if i < len(we.Cases)-1 {
				nextBlock = g.fn.NewBlock(BlockPlain)
			} else {
				nextBlock = mergeBlock
			}

			// Conditional branch
			g.block.Kind = BlockIf
			g.block.Control = cond
			g.block.AddSucc(thenBlock)
			g.block.AddSucc(nextBlock)

			// Seal then block (its only predecessor is the current conditional block)
			g.sealBlock(thenBlock)

			// Seal next block (its only predecessor is the current conditional block)
			g.sealBlock(nextBlock)

			// Generate then block
			g.block = thenBlock
			if err := g.generateStatement(c.Body); err != nil {
				return nil, err
			}

			// If expression, get result value
			if resultType != nil && g.block != nil {
				val := g.getLastValue()
				if val != nil {
					phiArgs = append(phiArgs, &PhiArg{From: g.block, Value: val})
				}
			}

			// Jump to merge
			if g.block != nil && g.block.Kind == BlockPlain {
				g.block.AddSucc(mergeBlock)
			}

			// Move to next condition block
			g.block = nextBlock
		}
	}

	// Seal merge block (all predecessors should now be connected)
	g.sealBlock(mergeBlock)

	// Continue in merge block
	g.block = mergeBlock

	// Create phi node if this is an expression
	if resultType != nil && len(phiArgs) > 0 {
		phi := g.block.NewPhiValue(resultType)
		phi.PhiArgs = phiArgs
		return phi, nil
	}

	return nil, nil
}

// getLastValue returns the last value added to the current block.
func (g *Generator) getLastValue() *Value {
	if g.block == nil || len(g.block.Values) == 0 {
		return nil
	}
	return g.block.Values[len(g.block.Values)-1]
}

// generateExpr generates IR for an expression.
func (g *Generator) generateExpr(expr semantic.TypedExpression) (*Value, error) {
	switch e := expr.(type) {
	case *semantic.TypedLiteralExpr:
		return g.generateLiteral(e)

	case *semantic.TypedIdentifierExpr:
		return g.generateIdentifier(e)

	case *semantic.TypedBinaryExpr:
		return g.generateBinary(e)

	case *semantic.TypedUnaryExpr:
		return g.generateUnary(e)

	case *semantic.TypedCallExpr:
		return g.generateCall(e)

	case *semantic.TypedFieldAccessExpr:
		return g.generateFieldAccess(e)

	case *semantic.TypedIndexExpr:
		return g.generateIndex(e)

	case *semantic.TypedLenExpr:
		return g.generateLen(e)

	case *semantic.TypedArrayLiteralExpr:
		return g.generateArrayLiteral(e)

	case *semantic.TypedStructLiteralExpr:
		return g.generateStructLiteral(e)

	case *semantic.TypedClassLiteralExpr:
		return g.generateClassLiteral(e)

	case *semantic.TypedMethodCallExpr:
		return g.generateMethodCall(e)

	case *semantic.TypedSafeCallExpr:
		return g.generateSafeCall(e)

	case *semantic.TypedSelfExpr:
		return g.generateSelf(e)

	case *semantic.TypedIfStmt:
		return g.generateIfExpr(e)

	case *semantic.TypedWhenExpr:
		return g.generateWhen(e)

	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// generateLiteral generates IR for a literal expression.
func (g *Generator) generateLiteral(le *semantic.TypedLiteralExpr) (*Value, error) {
	irType := g.convertType(le.Type)

	switch le.LitType {
	case ast.LiteralTypeInteger:
		var intVal int64

		// Check if the type is unsigned to use the correct parsing
		if intType, ok := irType.(*IntType); ok && !intType.Signed {
			// Parse as unsigned, then store bits as int64
			uval, err := strconv.ParseUint(le.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid unsigned integer literal: %s", le.Value)
			}
			intVal = int64(uval) // Store bits, interpretation depends on type
		} else {
			// Try parsing as signed first
			val, err := strconv.ParseInt(le.Value, 10, 64)
			if err != nil {
				// If signed parsing fails, try unsigned (for u64 max value etc.)
				uval, uerr := strconv.ParseUint(le.Value, 10, 64)
				if uerr != nil {
					return nil, fmt.Errorf("invalid integer literal: %s", le.Value)
				}
				intVal = int64(uval)
			} else {
				intVal = val
			}
		}

		v := g.block.NewValue(OpConst, irType)
		v.AuxInt = intVal
		return v, nil

	case ast.LiteralTypeFloat:
		var floatVal float64
		fmt.Sscanf(le.Value, "%f", &floatVal)

		v := g.block.NewValue(OpConst, irType)
		v.AuxFloat = floatVal
		return v, nil

	case ast.LiteralTypeString:
		// Add string to constant pool
		idx := g.prog.AddString(le.Value)

		v := g.block.NewValue(OpConst, irType)
		v.AuxInt = int64(idx)
		v.AuxString = le.Value
		return v, nil

	case ast.LiteralTypeBoolean:
		v := g.block.NewValue(OpConst, irType)
		if le.Value == "true" {
			v.AuxInt = 1
		} else {
			v.AuxInt = 0
		}
		return v, nil

	case ast.LiteralTypeNull:
		v := g.block.NewValue(OpWrapNull, irType)
		return v, nil

	default:
		return nil, fmt.Errorf("unknown literal type: %d", le.LitType)
	}
}

// generateIdentifier generates IR for an identifier expression.
func (g *Generator) generateIdentifier(ie *semantic.TypedIdentifierExpr) (*Value, error) {
	// Look up the current SSA definition
	return g.readVariable(ie.Name, g.block), nil
}

// generateBinary generates IR for a binary expression.
// Dispatches to specialized handlers for special operators.
func (g *Generator) generateBinary(be *semantic.TypedBinaryExpr) (*Value, error) {
	switch be.Op {
	case "&&", "||":
		return g.generateShortCircuit(be)
	case "?:":
		return g.generateElvis(be)
	case "==", "!=":
		if result, handled, err := g.tryGenerateNullComparison(be); handled {
			return result, err
		}
	}

	return g.generateBinaryOp(be)
}

// tryGenerateNullComparison handles `x == null` and `x != null` comparisons.
// Returns (result, true, nil) if one operand is a null literal.
// Returns (nil, false, nil) if neither operand is null (caller should use regular comparison).
func (g *Generator) tryGenerateNullComparison(be *semantic.TypedBinaryExpr) (*Value, bool, error) {
	leftIsNull := isNullLiteral(be.Left)
	rightIsNull := isNullLiteral(be.Right)

	if !leftIsNull && !rightIsNull {
		return nil, false, nil
	}

	// Determine which operand to test for null
	testExpr := be.Left
	if leftIsNull {
		testExpr = be.Right
	}

	val, err := g.generateExpr(testExpr)
	if err != nil {
		return nil, true, err
	}

	b := g.builder()
	isNullVal := b.IsNull(val)

	if be.Op == "!=" {
		return b.Not(isNullVal), true, nil
	}
	return isNullVal, true, nil
}

// generateBinaryOp generates IR for standard binary operations (+, -, *, /, etc).
func (g *Generator) generateBinaryOp(be *semantic.TypedBinaryExpr) (*Value, error) {
	left, err := g.generateExpr(be.Left)
	if err != nil {
		return nil, err
	}

	right, err := g.generateExpr(be.Right)
	if err != nil {
		return nil, err
	}

	op, ok := binaryOpMap[be.Op]
	if !ok {
		return nil, fmt.Errorf("unknown binary operator: %s", be.Op)
	}

	resultType := g.convertType(be.Type)
	v := g.block.NewValue(op, resultType)
	v.AddArg(left)
	v.AddArg(right)
	return v, nil
}

// generateShortCircuit generates IR for && and || with short-circuit evaluation.
func (g *Generator) generateShortCircuit(be *semantic.TypedBinaryExpr) (*Value, error) {
	// Generate left operand
	left, err := g.generateExpr(be.Left)
	if err != nil {
		return nil, err
	}

	// Create blocks for short-circuit
	rightBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Set up branch based on operator
	g.block.Kind = BlockIf
	g.block.Control = left
	leftBlock := g.block

	if be.Op == "&&" {
		// AND: if left is false, skip right (result is false)
		g.block.AddSucc(rightBlock) // true -> evaluate right
		g.block.AddSucc(mergeBlock) // false -> skip
	} else {
		// OR: if left is true, skip right (result is true)
		g.block.AddSucc(mergeBlock) // true -> skip
		g.block.AddSucc(rightBlock) // false -> evaluate right
	}

	// Generate right operand
	g.block = rightBlock
	right, err := g.generateExpr(be.Right)
	if err != nil {
		return nil, err
	}
	rightBlock = g.block // May have changed
	rightBlock.AddSucc(mergeBlock)

	// Create phi in merge block
	g.block = mergeBlock

	phi := g.block.NewPhiValue(TypeBool)
	if be.Op == "&&" {
		// AND: false from left, right's value from right
		falseVal := g.block.NewValue(OpConst, TypeBool)
		falseVal.AuxInt = 0
		phi.PhiArgs = []*PhiArg{
			{From: leftBlock, Value: falseVal},
			{From: rightBlock, Value: right},
		}
	} else {
		// OR: true from left, right's value from right
		trueVal := g.block.NewValue(OpConst, TypeBool)
		trueVal.AuxInt = 1
		phi.PhiArgs = []*PhiArg{
			{From: leftBlock, Value: trueVal},
			{From: rightBlock, Value: right},
		}
	}

	return phi, nil
}

// generateElvis generates IR for the elvis operator (a ?: b).
// Returns a if a is not null, otherwise returns b.
func (g *Generator) generateElvis(be *semantic.TypedBinaryExpr) (*Value, error) {
	// Generate left operand
	left, err := g.generateExpr(be.Left)
	if err != nil {
		return nil, err
	}

	// Create blocks for short-circuit
	rightBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Check if left is null
	isNull := g.block.NewValue(OpIsNull, TypeBool)
	isNull.AddArg(left)

	// Set up branch: if null, go to right block; otherwise go to merge
	g.block.Kind = BlockIf
	g.block.Control = isNull
	leftBlock := g.block
	g.block.AddSucc(rightBlock) // null -> evaluate right
	g.block.AddSucc(mergeBlock) // not null -> skip

	// Seal right block before generating it
	g.sealBlock(rightBlock)

	// Generate right operand
	g.block = rightBlock
	right, err := g.generateExpr(be.Right)
	if err != nil {
		return nil, err
	}
	rightBlock = g.block // May have changed
	rightBlock.AddSucc(mergeBlock)

	// Seal merge block now that all predecessors are known
	g.sealBlock(mergeBlock)

	// Create phi in merge block
	g.block = mergeBlock

	resultType := g.convertType(be.Type)
	phi := g.block.NewPhiValue(resultType)

	// If left was not null (from leftBlock), unwrap it
	// If left was null (from rightBlock), use right value
	unwrapped := leftBlock.NewValue(OpUnwrap, resultType)
	unwrapped.AddArg(left)

	phi.PhiArgs = []*PhiArg{
		{From: leftBlock, Value: unwrapped},
		{From: rightBlock, Value: right},
	}

	return phi, nil
}

// generateUnary generates IR for a unary expression.
func (g *Generator) generateUnary(ue *semantic.TypedUnaryExpr) (*Value, error) {
	operand, err := g.generateExpr(ue.Operand)
	if err != nil {
		return nil, err
	}

	b := g.builder()
	resultType := g.convertType(ue.Type)

	switch ue.Op {
	case "!":
		return b.Not(operand), nil
	case "-":
		return b.Neg(resultType, operand), nil
	default:
		return nil, fmt.Errorf("unknown unary operator: %s", ue.Op)
	}
}

// generateCall generates IR for a function call.
func (g *Generator) generateCall(ce *semantic.TypedCallExpr) (*Value, error) {
	// Generate arguments
	var args []*Value
	for _, arg := range ce.Arguments {
		// If this argument is an identifier that's an owned pointer,
		// mark it as moved (ownership is transferring to the callee)
		if ident, ok := arg.(*semantic.TypedIdentifierExpr); ok {
			if elemType, _ := g.getOwnedPointerInfo(ident.Type); elemType != nil {
				g.markMoved(ident.Name)
			}
		}

		v, err := g.generateExpr(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}

	resultType := g.convertType(ce.Type)
	return g.builder().Call(ce.Name, resultType, args...), nil
}

// generateFieldAccess generates IR for field access.
func (g *Generator) generateFieldAccess(fa *semantic.TypedFieldAccessExpr) (*Value, error) {
	// Generate object
	obj, err := g.generateExpr(fa.Object)
	if err != nil {
		return nil, err
	}

	// Get field offset
	offset := g.getFieldOffset(fa.Object.GetType(), fa.Field)

	resultType := g.convertType(fa.Type)

	// Create field pointer
	fieldPtr := g.block.NewValue(OpFieldPtr, &PtrType{Elem: resultType})
	fieldPtr.AddArg(obj)
	fieldPtr.AuxInt = int64(offset)

	// For struct/class types embedded by value, return the pointer directly
	// (the caller will use it for further field access or as needed)
	// Only load primitive values (int, bool, string) and pointers
	switch resultType.(type) {
	case *StructType:
		// Return pointer to embedded struct, don't load
		return fieldPtr, nil
	default:
		// Load primitive value
		load := g.block.NewValue(OpLoad, resultType)
		load.AddArg(fieldPtr)
		return load, nil
	}
}

// generateIndex generates IR for array index access.
func (g *Generator) generateIndex(ie *semantic.TypedIndexExpr) (*Value, error) {
	// Generate array and index
	arr, err := g.generateExpr(ie.Array)
	if err != nil {
		return nil, err
	}

	idx, err := g.generateExpr(ie.Index)
	if err != nil {
		return nil, err
	}

	resultType := g.convertType(ie.Type)

	// Create index pointer
	elemPtr := g.block.NewValue(OpIndexPtr, &PtrType{Elem: resultType})
	elemPtr.AddArg(arr)
	elemPtr.AddArg(idx)

	// Load value
	load := g.block.NewValue(OpLoad, resultType)
	load.AddArg(elemPtr)

	return load, nil
}

// generateLen generates IR for len() builtin.
func (g *Generator) generateLen(le *semantic.TypedLenExpr) (*Value, error) {
	// Array size is known at compile time
	v := g.block.NewValue(OpConst, TypeS64)
	v.AuxInt = int64(le.ArraySize)
	return v, nil
}

// generateArrayLiteral generates IR for an array literal.
func (g *Generator) generateArrayLiteral(al *semantic.TypedArrayLiteralExpr) (*Value, error) {
	elemType := g.convertType(al.Type.ElementType)
	arrayType := &ArrayType{Elem: elemType, Len: al.Type.Size}
	b := g.builder()

	// Allocate space
	alloc := b.Alloc(arrayType, int64(arrayType.Size()))

	// Store each element
	for i, elemExpr := range al.Elements {
		elem, err := g.generateExpr(elemExpr)
		if err != nil {
			return nil, err
		}

		idx := b.ConstInt(TypeS64, int64(i))
		elemPtr := b.IndexPtr(alloc, idx, elemType)
		b.Store(elemPtr, elem)
	}

	return alloc, nil
}

// generateStructLiteral generates IR for a struct literal.
func (g *Generator) generateStructLiteral(sl *semantic.TypedStructLiteralExpr) (*Value, error) {
	structType := g.convertStructType(&sl.Type)
	b := g.builder()

	// Allocate space
	alloc := b.Alloc(structType, int64(structType.Size()))

	// Store each field
	for i, argExpr := range sl.Args {
		fieldType := structType.Fields[i].Type
		fieldOffset := int64(structType.Fields[i].Offset)

		// If this argument is an identifier that's an owned pointer,
		// mark it as moved (ownership is transferring to the struct)
		if ident, ok := argExpr.(*semantic.TypedIdentifierExpr); ok {
			if elemType, _ := g.getOwnedPointerInfo(ident.Type); elemType != nil {
				g.markMoved(ident.Name)
			}
		}

		// Generate field value with null/nullable handling
		arg, err := g.generateTypedValue(argExpr, fieldType)
		if err != nil {
			return nil, err
		}

		fieldPtr := b.FieldPtr(alloc, fieldType, fieldOffset)

		// For embedded struct fields, copy the data instead of storing a pointer
		if _, isStruct := fieldType.(*StructType); isStruct {
			b.MemCopy(fieldPtr, arg, int64(fieldType.Size()))
		} else {
			b.Store(fieldPtr, arg)
		}
	}

	return alloc, nil
}

// generateClassLiteral generates IR for a class literal.
func (g *Generator) generateClassLiteral(cl *semantic.TypedClassLiteralExpr) (*Value, error) {
	classType := g.convertClassType(&cl.Type)
	b := g.builder()

	// Allocate space
	alloc := b.Alloc(classType, int64(classType.Size()))

	// Store each field
	for i, argExpr := range cl.Args {
		fieldType := classType.Fields[i].Type
		fieldOffset := int64(classType.Fields[i].Offset)

		// If this argument is an identifier that's an owned pointer,
		// mark it as moved (ownership is transferring to the class)
		if ident, ok := argExpr.(*semantic.TypedIdentifierExpr); ok {
			if elemType, _ := g.getOwnedPointerInfo(ident.Type); elemType != nil {
				g.markMoved(ident.Name)
			}
		}

		// Generate field value with null/nullable handling
		arg, err := g.generateTypedValue(argExpr, fieldType)
		if err != nil {
			return nil, err
		}

		fieldPtr := b.FieldPtr(alloc, fieldType, fieldOffset)
		b.Store(fieldPtr, arg)
	}

	return alloc, nil
}

// generateMethodCall generates IR for a method call.
func (g *Generator) generateMethodCall(mc *semantic.TypedMethodCallExpr) (*Value, error) {
	// Special handling for Heap.new
	if mc.Method == "new" {
		// Check if it's Heap.new (object is Heap identifier)
		if ident, ok := mc.Object.(*semantic.TypedIdentifierExpr); ok && ident.Name == "Heap" {
			return g.generateHeapNew(mc)
		}
	}

	// Special handling for .copy()
	if mc.Method == "copy" && len(mc.Arguments) == 0 {
		return g.generateCopy(mc)
	}

	// Handle safe navigation (?.method())
	if mc.SafeNavigation {
		return g.generateSafeMethodCall(mc)
	}

	// Generate object (becomes first argument for instance methods)
	obj, err := g.generateExpr(mc.Object)
	if err != nil {
		return nil, err
	}

	// Generate other arguments
	var args []*Value
	if mc.ResolvedMethod != nil && !mc.ResolvedMethod.IsStatic {
		args = append(args, obj)
	}
	for _, arg := range mc.Arguments {
		v, err := g.generateExpr(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}

	// Determine mangled function name (include param count for overloading)
	className := g.getTypeName(mc.Object.GetType())
	// Count params: self (for instance methods) + explicit arguments
	paramCount := len(mc.Arguments)
	if mc.ResolvedMethod != nil && !mc.ResolvedMethod.IsStatic {
		paramCount++ // Count self parameter
	}
	mangledName := fmt.Sprintf("%s_%s_%d", className, mc.Method, paramCount)

	resultType := g.convertType(mc.Type)

	call := g.block.NewValue(OpCall, resultType)
	call.AuxString = mangledName
	for _, arg := range args {
		call.AddArg(arg)
	}

	return call, nil
}

// generateSafeMethodCall generates IR for safe method call (?.).
// If the object is null, returns null; otherwise calls the method.
func (g *Generator) generateSafeMethodCall(mc *semantic.TypedMethodCallExpr) (*Value, error) {
	obj, err := g.generateExpr(mc.Object)
	if err != nil {
		return nil, err
	}

	resultType := g.convertType(mc.Type)
	blocks := g.createNullCheckBlocks()

	// Branch on null check
	g.emitNullCheck(obj, blocks)

	// Null path: return wrapped null
	g.block = blocks.nullBlock
	nullResult := g.block.NewValue(OpWrapNull, resultType)
	blocks.nullBlock.AddSucc(blocks.mergeBlock)

	// Not-null path: unwrap, call method, wrap result
	g.block = blocks.notNullBlock
	wrapped, err := g.generateUnwrappedMethodCall(obj, mc, resultType)
	if err != nil {
		return nil, err
	}
	blocks.notNullBlock = g.block // block may have changed during arg generation
	blocks.notNullBlock.AddSucc(blocks.mergeBlock)

	// Merge results with phi node
	return g.mergeNullCheckResults(blocks, resultType, nullResult, wrapped)
}

// nullCheckBlocks holds the basic blocks for a null-check pattern.
type nullCheckBlocks struct {
	nullBlock    *Block
	notNullBlock *Block
	mergeBlock   *Block
}

// createNullCheckBlocks creates the three blocks needed for null-check control flow.
func (g *Generator) createNullCheckBlocks() *nullCheckBlocks {
	return &nullCheckBlocks{
		nullBlock:    g.fn.NewBlock(BlockPlain),
		notNullBlock: g.fn.NewBlock(BlockPlain),
		mergeBlock:   g.fn.NewBlock(BlockPlain),
	}
}

// emitNullCheck generates IR to branch based on whether a value is null.
func (g *Generator) emitNullCheck(val *Value, blocks *nullCheckBlocks) {
	isNull := g.block.NewValue(OpIsNull, TypeBool)
	isNull.AddArg(val)

	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(blocks.nullBlock)
	g.block.AddSucc(blocks.notNullBlock)

	g.sealBlock(blocks.nullBlock)
	g.sealBlock(blocks.notNullBlock)
}

// generateUnwrappedMethodCall generates a method call on an unwrapped nullable value.
func (g *Generator) generateUnwrappedMethodCall(
	obj *Value,
	mc *semantic.TypedMethodCallExpr,
	resultType Type,
) (*Value, error) {
	// Unwrap the nullable object
	innerType := UnwrapNullableType(mc.Object.GetType())
	unwrappedIRType := g.convertType(innerType)
	unwrapped := g.block.NewValue(OpUnwrap, unwrappedIRType)
	unwrapped.AddArg(obj)

	// Build argument list
	args, err := g.buildMethodArgs(unwrapped, mc)
	if err != nil {
		return nil, err
	}

	// Generate the call
	mangledName := g.mangleMethodName(innerType, mc)
	callResultType := UnwrapIRNullableType(resultType)

	call := g.block.NewValue(OpCall, callResultType)
	call.AuxString = mangledName
	for _, arg := range args {
		call.AddArg(arg)
	}

	// Wrap result as nullable
	wrapped := g.block.NewValue(OpWrap, resultType)
	wrapped.AddArg(call)
	return wrapped, nil
}

// buildMethodArgs builds the argument list for a method call.
func (g *Generator) buildMethodArgs(receiver *Value, mc *semantic.TypedMethodCallExpr) ([]*Value, error) {
	var args []*Value

	// Add receiver for instance methods
	if mc.ResolvedMethod != nil && !mc.ResolvedMethod.IsStatic {
		args = append(args, receiver)
	}

	// Add explicit arguments
	for _, arg := range mc.Arguments {
		v, err := g.generateExpr(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}

	return args, nil
}

// mangleMethodName generates the mangled name for a method call.
func (g *Generator) mangleMethodName(receiverType semantic.Type, mc *semantic.TypedMethodCallExpr) string {
	className := g.getTypeName(receiverType)
	paramCount := len(mc.Arguments)
	if mc.ResolvedMethod != nil && !mc.ResolvedMethod.IsStatic {
		paramCount++ // Count self parameter
	}
	return fmt.Sprintf("%s_%s_%d", className, mc.Method, paramCount)
}

// mergeNullCheckResults creates a phi node to merge null and non-null paths.
func (g *Generator) mergeNullCheckResults(
	blocks *nullCheckBlocks,
	resultType Type,
	nullResult, notNullResult *Value,
) (*Value, error) {
	g.sealBlock(blocks.mergeBlock)
	g.block = blocks.mergeBlock

	phi := g.block.NewPhiValue(resultType)
	phi.PhiArgs = []*PhiArg{
		{From: blocks.nullBlock, Value: nullResult},
		{From: blocks.notNullBlock, Value: notNullResult},
	}

	return phi, nil
}

// generateHeapNew generates IR for Heap.new(value).
func (g *Generator) generateHeapNew(mc *semantic.TypedMethodCallExpr) (*Value, error) {
	if len(mc.Arguments) != 1 {
		return nil, fmt.Errorf("Heap.new requires exactly 1 argument")
	}

	arg := mc.Arguments[0]

	// If the argument is a struct or class literal, it already allocates on heap.
	// Just generate it directly and return the pointer.
	switch arg.(type) {
	case *semantic.TypedStructLiteralExpr, *semantic.TypedClassLiteralExpr:
		return g.generateExpr(arg)
	}

	// For other expressions, we need to allocate and copy.
	// Generate the value to allocate
	val, err := g.generateExpr(arg)
	if err != nil {
		return nil, err
	}

	// Get size of the value
	valType := val.Type
	size := valType.Size()

	// Allocate on heap
	alloc := g.block.NewValue(OpAlloc, &PtrType{Elem: valType})
	alloc.AuxInt = int64(size)

	// Store initial value
	store := g.block.NewValue(OpStore, nil)
	store.AddArg(alloc)
	store.AddArg(val)

	return alloc, nil
}

// generateCopy generates IR for .copy() method.
func (g *Generator) generateCopy(mc *semantic.TypedMethodCallExpr) (*Value, error) {
	// Generate object to copy
	obj, err := g.generateExpr(mc.Object)
	if err != nil {
		return nil, err
	}

	// Create deep copy
	copy := g.block.NewValue(OpCopy, obj.Type)
	copy.AddArg(obj)

	return copy, nil
}

// generateSafeCall generates IR for safe navigation (?.).
func (g *Generator) generateSafeCall(sc *semantic.TypedSafeCallExpr) (*Value, error) {
	// Generate object
	obj, err := g.generateExpr(sc.Object)
	if err != nil {
		return nil, err
	}

	resultType := g.convertType(sc.Type)

	// Check if null
	isNull := g.block.NewValue(OpIsNull, TypeBool)
	isNull.AddArg(obj)

	// Create blocks
	notNullBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Create null result in current block (before branching)
	// This value will be used by the phi when we take the null path
	nullResult := g.block.NewValue(OpWrapNull, resultType)

	// Branch
	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(mergeBlock) // is null -> skip
	g.block.AddSucc(notNullBlock)
	nullBlock := g.block

	// Not null path: access field
	g.block = notNullBlock

	// Unwrap nullable
	// sc.InnerType is the struct type when ThroughPointer is true (pointer was auto-dereferenced)
	// We need the actual unwrapped type which includes the pointer wrapper
	innerType := g.convertType(sc.InnerType)
	unwrapType := innerType
	if sc.ThroughPointer {
		// When accessing through pointer (e.g., *TreeNode?), unwrap gives *TreeNode
		unwrapType = &PtrType{Elem: innerType}
	}
	unwrapped := g.block.NewValue(OpUnwrap, unwrapType)
	unwrapped.AddArg(obj)

	// Get field - FieldPtr operates on the pointer
	fieldPtr := g.block.NewValue(OpFieldPtr, &PtrType{Elem: resultType})
	fieldPtr.AddArg(unwrapped)
	fieldPtr.AuxInt = int64(sc.FieldOffset)

	fieldVal := g.block.NewValue(OpLoad, resultType)
	fieldVal.AddArg(fieldPtr)

	// Wrap result in nullable
	wrapped := g.block.NewValue(OpWrap, resultType)
	wrapped.AddArg(fieldVal)

	notNullBlock.AddSucc(mergeBlock)

	// Merge block
	g.block = mergeBlock

	phi := g.block.NewPhiValue(resultType)
	phi.PhiArgs = []*PhiArg{
		{From: nullBlock, Value: nullResult},
		{From: notNullBlock, Value: wrapped},
	}

	return phi, nil
}

// generateSelf generates IR for self expression.
func (g *Generator) generateSelf(_ *semantic.TypedSelfExpr) (*Value, error) {
	// self is the first parameter
	return g.readVariable("self", g.block), nil
}

// generateIfExpr generates IR for an if expression (with result).
func (g *Generator) generateIfExpr(is *semantic.TypedIfStmt) (*Value, error) {
	if is.ResultType == nil {
		// Not an expression, generate as statement
		return nil, g.generateIf(is)
	}

	// Generate condition
	cond, err := g.generateExpr(is.Condition)
	if err != nil {
		return nil, err
	}

	resultType := g.convertType(is.ResultType)

	// Create blocks
	thenBlock := g.fn.NewBlock(BlockPlain)
	elseBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Branch
	g.block.Kind = BlockIf
	g.block.Control = cond
	g.block.AddSucc(thenBlock)
	g.block.AddSucc(elseBlock)

	// Generate then
	g.block = thenBlock
	if err := g.generateBlock(is.ThenBranch); err != nil {
		return nil, err
	}
	thenVal := g.getLastValue()
	thenEndBlock := g.block
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(mergeBlock)
	}

	// Generate else
	g.block = elseBlock
	if err := g.generateStatement(is.ElseBranch); err != nil {
		return nil, err
	}
	elseVal := g.getLastValue()
	elseEndBlock := g.block
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(mergeBlock)
	}

	// Merge
	g.block = mergeBlock

	phi := g.block.NewPhiValue(resultType)
	phi.PhiArgs = []*PhiArg{
		{From: thenEndBlock, Value: thenVal},
		{From: elseEndBlock, Value: elseVal},
	}

	return phi, nil
}

// ============================================================================
// SSA Construction
// ============================================================================

// writeVariable records a definition of a variable in a block.
// Delegates to SSABuilder.
func (g *Generator) writeVariable(name string, block *Block, val *Value) {
	g.ssa.WriteVariable(name, block, val)
}

// readVariable returns the current definition of a variable.
// Delegates to SSABuilder.
func (g *Generator) readVariable(name string, block *Block) *Value {
	return g.ssa.ReadVariable(name, block)
}

// sealBlock marks a block as sealed (no more predecessors will be added).
// Delegates to SSABuilder.
func (g *Generator) sealBlock(block *Block) {
	g.ssa.SealBlock(block)
}

// ============================================================================
// Memory Management
// ============================================================================

// emitFreeIfOwned emits an OpFree if the variable holds an owned pointer.
// For nullable owned pointers, it checks for null before freeing.
func (g *Generator) emitFreeIfOwned(name string, semType semantic.Type) {
	// Get the owned pointer element type and size
	elemType, size := g.getOwnedPointerInfo(semType)
	if elemType == nil {
		return // Not an owned pointer
	}

	// Check if variable is defined on all paths to current block
	// Variables defined only in loops/conditionals may not be available here
	if !g.ssa.IsVariableDefinedOnAllPaths(name, g.block) {
		return // Variable not defined on all paths, skip cleanup
	}

	// Read the current value
	oldVal := g.readVariable(name, g.block)
	if oldVal == nil {
		return
	}

	// Check if this is a nullable type
	if isNullableOwnedType(semType) {
		// For nullable, check if non-null before freeing
		freeBlock := g.fn.NewBlock(BlockPlain)
		continueBlock := g.fn.NewBlock(BlockPlain)

		// Check if null
		isNull := g.block.NewValue(OpIsNull, TypeBool, oldVal)

		// Branch: if null -> continue, if not null -> free
		g.block.Kind = BlockIf
		g.block.Control = isNull
		g.block.AddSucc(continueBlock) // null -> skip free
		g.block.AddSucc(freeBlock)     // not null -> free

		// Free block - unwrap and recursively free
		g.block = freeBlock
		unwrapped := g.block.NewValue(OpUnwrap, &PtrType{Elem: g.convertType(elemType)}, oldVal)
		g.emitRecursiveFree(unwrapped, elemType, size)
		g.block.AddSucc(continueBlock)
		g.sealBlock(freeBlock)

		// Continue block
		g.block = continueBlock
		g.sealBlock(continueBlock)
	} else {
		// Non-nullable owned pointer - recursively free it
		g.emitRecursiveFree(oldVal, elemType, size)
	}
}

// emitRecursiveFree frees a struct and all its owned pointer fields recursively.
// ptr is the pointer to the struct, elemType is the semantic type of the element,
// and size is the byte size of the struct.
// The visiting map tracks types currently being processed to detect self-referential types.
func (g *Generator) emitRecursiveFree(ptr *Value, elemType semantic.Type, size int) {
	g.emitRecursiveFreeWithVisited(ptr, elemType, size, make(map[string]bool))
}

func (g *Generator) emitRecursiveFreeWithVisited(ptr *Value, elemType semantic.Type, size int, visiting map[string]bool) {
	// First, recursively free any owned pointer fields in this struct
	if st := g.getSemanticStructType(elemType); st != nil {
		// Check if this is a self-referential type (already being processed)
		typeName := st.Name
		if visiting[typeName] {
			// Self-referential type - generate a runtime loop instead of compile-time recursion
			g.emitRuntimeFreeLoop(ptr, st, size)
			return
		}

		// Mark as visiting
		visiting[typeName] = true
		defer func() { delete(visiting, typeName) }()

		for _, field := range st.Fields {
			if ownedElemType, fieldSize := g.getOwnedPointerInfo(field.Type); ownedElemType != nil {
				// This field is an owned pointer - need to free it
				offset := st.FieldOffset(field.Name)
				irElemType := g.convertType(ownedElemType)

				// Get pointer to the field
				fieldPtrType := &PtrType{Elem: g.convertType(field.Type)}
				fieldPtr := g.block.NewValue(OpFieldPtr, fieldPtrType, ptr)
				fieldPtr.AuxInt = int64(offset)

				// Load the field value
				fieldVal := g.block.NewValue(OpLoad, g.convertType(field.Type), fieldPtr)

				// Check if this is a nullable owned pointer
				if isNullableOwnedType(field.Type) {
					// Need to check for null before freeing
					freeFieldBlock := g.fn.NewBlock(BlockPlain)
					continueFieldBlock := g.fn.NewBlock(BlockPlain)

					isNull := g.block.NewValue(OpIsNull, TypeBool, fieldVal)
					g.block.Kind = BlockIf
					g.block.Control = isNull
					g.block.AddSucc(continueFieldBlock) // null -> skip
					g.block.AddSucc(freeFieldBlock)     // not null -> free

					// Free field block
					g.block = freeFieldBlock
					unwrapped := g.block.NewValue(OpUnwrap, &PtrType{Elem: irElemType}, fieldVal)
					g.emitRecursiveFreeWithVisited(unwrapped, ownedElemType, fieldSize, visiting)
					g.block.AddSucc(continueFieldBlock)
					g.sealBlock(freeFieldBlock)

					// Continue block
					g.block = continueFieldBlock
					g.sealBlock(continueFieldBlock)
				} else {
					// Non-nullable owned pointer field - just free it
					g.emitRecursiveFreeWithVisited(fieldVal, ownedElemType, fieldSize, visiting)
				}
			}
		}
	}

	// Now free the struct itself
	freeVal := g.block.NewValue(OpFree, nil, ptr)
	freeVal.AuxInt = int64(size)
}

// emitRuntimeFreeLoop generates a runtime loop to free a self-referential linked structure.
// This handles types like linked lists where the struct contains a pointer to the same type.
func (g *Generator) emitRuntimeFreeLoop(ptr *Value, st *semantic.StructType, size int) {
	// Look up the IR struct by name to get accurate field info
	// (the semantic st may have stale/empty fields due to two-pass registration)
	irStruct := g.prog.StructByName(st.Name)
	if irStruct == nil || len(irStruct.Fields) == 0 {
		// Can't find struct info, just free without recursion
		freeVal := g.block.NewValue(OpFree, nil, ptr)
		freeVal.AuxInt = int64(size)
		return
	}

	// Find the self-referential owned pointer field
	var selfRefField *StructField
	var selfRefOffset int
	for i := range irStruct.Fields {
		field := &irStruct.Fields[i]
		// Check if this field is a nullable pointer to the same struct type
		if nt, ok := field.Type.(*NullableType); ok {
			if pt, ok := nt.Elem.(*PtrType); ok {
				if innerSt, ok := pt.Elem.(*StructType); ok && innerSt.Name == st.Name {
					selfRefField = field
					selfRefOffset = field.Offset
					break
				}
			}
		}
		// Also check non-nullable pointer
		if pt, ok := field.Type.(*PtrType); ok {
			if innerSt, ok := pt.Elem.(*StructType); ok && innerSt.Name == st.Name {
				selfRefField = field
				selfRefOffset = field.Offset
				break
			}
		}
	}

	if selfRefField == nil {
		// No self-referential field found, just free normally
		freeVal := g.block.NewValue(OpFree, nil, ptr)
		freeVal.AuxInt = int64(size)
		return
	}

	// Generate a runtime loop (following the while loop pattern):
	// current = ptr (wrapped as nullable)
	// while (current != null) {
	//     unwrapped = unwrap(current)
	//     next = unwrapped.selfRefField
	//     free(unwrapped)
	//     current = next
	// }

	// Use a unique variable name for this loop's current pointer
	loopVarName := fmt.Sprintf("__free_loop_%d", g.fn.NumBlocks())

	loopHeader := g.fn.NewBlock(BlockPlain)
	loopBody := g.fn.NewBlock(BlockPlain)
	loopExit := g.fn.NewBlock(BlockPlain)

	// Use the field type directly (it's already nullable)
	loopVarType := selfRefField.Type

	// Wrap the initial pointer as nullable to match the field type
	irPtrType := &PtrType{Elem: irStruct}
	nullablePtrType := &NullableType{Elem: irPtrType}
	wrappedPtr := g.block.NewValue(OpWrap, nullablePtrType, ptr)

	// Write initial value to loop variable
	g.writeVariable(loopVarName, g.block, wrappedPtr)

	// Jump to header
	g.block.AddSucc(loopHeader)
	g.block = loopHeader

	// Read current pointer (SSA builder will create phi if needed)
	currentPtr := g.readVariable(loopVarName, g.block)

	// Check if null
	isNull := g.block.NewValue(OpIsNull, TypeBool, currentPtr)
	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(loopExit)  // null -> exit
	g.block.AddSucc(loopBody)  // not null -> body

	// Seal body block - only predecessor is header
	g.sealBlock(loopBody)

	// Loop body
	g.block = loopBody

	// Unwrap to get actual pointer
	unwrapped := g.block.NewValue(OpUnwrap, irPtrType, currentPtr)

	// Load the next pointer (self-referential field)
	nextFieldPtrType := &PtrType{Elem: loopVarType}
	nextFieldPtr := g.block.NewValue(OpFieldPtr, nextFieldPtrType, unwrapped)
	nextFieldPtr.AuxInt = int64(selfRefOffset)
	nextVal := g.block.NewValue(OpLoad, loopVarType, nextFieldPtr)

	// Free current node
	freeVal := g.block.NewValue(OpFree, nil, unwrapped)
	freeVal.AuxInt = int64(size)

	// Update loop variable for next iteration
	g.writeVariable(loopVarName, g.block, nextVal)

	// Jump back to header
	g.block.AddSucc(loopHeader)

	// Seal header - now all predecessors are known (entry + body)
	g.sealBlock(loopHeader)

	// Seal exit block - only predecessor is header
	g.sealBlock(loopExit)

	// Continue after loop
	g.block = loopExit
}

// getSemanticStructType extracts StructType from a semantic type.
func (g *Generator) getSemanticStructType(t semantic.Type) *semantic.StructType {
	switch ty := t.(type) {
	case *semantic.StructType:
		return ty
	case semantic.StructType:
		return &ty
	default:
		return nil
	}
}

// getOwnedPointerInfo returns the element type and size if t is an owned pointer.
// Returns (nil, 0) if not an owned pointer.
func (g *Generator) getOwnedPointerInfo(t semantic.Type) (semantic.Type, int) {
	switch ty := t.(type) {
	case *semantic.OwnedPointerType:
		return ty.ElementType, g.getElementTypeSize(ty.ElementType)
	case semantic.OwnedPointerType:
		return ty.ElementType, g.getElementTypeSize(ty.ElementType)
	case *semantic.NullableType:
		// Check if inner type is owned pointer
		return g.getOwnedPointerInfo(ty.InnerType)
	case semantic.NullableType:
		return g.getOwnedPointerInfo(ty.InnerType)
	default:
		return nil, 0
	}
}

// getElementTypeSize computes the size of a type, looking up structs by name
// to avoid stale type copies from the two-pass registration in semantic analysis.
func (g *Generator) getElementTypeSize(t semantic.Type) int {
	// For struct types, look up by name to get the version with fields
	if st := g.getSemanticStructType(t); st != nil {
		if irStruct := g.prog.StructByName(st.Name); irStruct != nil {
			return irStruct.Size()
		}
	}
	// Fallback to direct conversion (for non-struct types)
	return g.convertType(t).Size()
}

// isNullableOwnedType checks if a type is a nullable owned pointer.
func isNullableOwnedType(t semantic.Type) bool {
	switch ty := t.(type) {
	case *semantic.NullableType:
		_, isOwned := ty.InnerType.(*semantic.OwnedPointerType)
		if !isOwned {
			_, isOwned = ty.InnerType.(semantic.OwnedPointerType)
		}
		return isOwned
	case semantic.NullableType:
		_, isOwned := ty.InnerType.(*semantic.OwnedPointerType)
		if !isOwned {
			_, isOwned = ty.InnerType.(semantic.OwnedPointerType)
		}
		return isOwned
	default:
		return false
	}
}

// ============================================================================
// Type Conversion
// ============================================================================

// convertType converts a semantic type to an IR type.
// This is a convenience wrapper around TypeConverter.Convert.
func (g *Generator) convertType(t semantic.Type) Type {
	return g.types().Convert(t)
}

// convertStructType converts a semantic struct type to IR.
// This is a convenience wrapper around TypeConverter.convertStruct.
func (g *Generator) convertStructType(st *semantic.StructType) *StructType {
	return g.types().convertStruct(st)
}

// convertClassType converts a semantic class type to IR struct.
// This is a convenience wrapper around TypeConverter.convertClass.
func (g *Generator) convertClassType(ct *semantic.ClassType) *StructType {
	return g.types().convertClass(ct)
}

// getFieldOffset returns the byte offset of a field in a type.
// This is a convenience wrapper around TypeConverter.FieldOffset.
func (g *Generator) getFieldOffset(t semantic.Type, fieldName string) int {
	return g.types().FieldOffset(t, fieldName)
}

// getTypeName returns the name of a type for method mangling.
// This is a convenience wrapper around TypeConverter.TypeName.
func (g *Generator) getTypeName(t semantic.Type) string {
	return g.types().TypeName(t)
}

