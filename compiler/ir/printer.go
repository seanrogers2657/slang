package ir

import (
	"fmt"
	"io"
	"strings"
)

// Printer outputs IR in a human-readable format.
type Printer struct {
	w      io.Writer
	indent int
}

// NewPrinter creates a new IR printer that writes to w.
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w}
}

// PrintProgram prints an entire program.
func (p *Printer) PrintProgram(prog *Program) {
	// Print struct definitions
	for _, s := range prog.Structs {
		p.PrintStruct(s)
		p.writeLine("")
	}

	// Print global variables
	for _, g := range prog.Globals {
		p.PrintGlobal(g)
	}
	if len(prog.Globals) > 0 {
		p.writeLine("")
	}

	// Print string constants
	if len(prog.Strings) > 0 {
		p.writeLine("// String constants")
		for i, s := range prog.Strings {
			p.writeLine("// str%d = %q", i, s)
		}
		p.writeLine("")
	}

	// Print functions
	for i, f := range prog.Functions {
		if i > 0 {
			p.writeLine("")
		}
		p.PrintFunction(f)
	}
}

// PrintStruct prints a struct type definition.
func (p *Printer) PrintStruct(s *StructType) {
	p.writeLine("type %s struct { // size=%d, align=%d", s.Name, s.Size(), s.Align())
	for _, f := range s.Fields {
		p.writeLine("    %s: %s // offset=%d", f.Name, f.Type.String(), f.Offset)
	}
	p.writeLine("}")
}

// PrintGlobal prints a global variable.
func (p *Printer) PrintGlobal(g *Global) {
	if g.Init != nil {
		p.writeLine("global %s: %s = %s", g.Name, g.Type.String(), g.Init.LongString())
	} else {
		p.writeLine("global %s: %s", g.Name, g.Type.String())
	}
}

// PrintFunction prints a function definition.
func (p *Printer) PrintFunction(f *Function) {
	// Function signature
	var params []string
	for _, param := range f.Params {
		params = append(params, fmt.Sprintf("v%d: %s", param.ID, param.Type.String()))
	}

	retStr := ""
	if f.ReturnType != nil && !f.ReturnType.Equal(TypeVoid) {
		retStr = " -> " + f.ReturnType.String()
	}

	p.writeLine("func %s(%s)%s {", f.Name, strings.Join(params, ", "), retStr)

	// Print each block
	for _, b := range f.Blocks {
		p.PrintBlock(b)
	}

	p.writeLine("}")
}

// PrintBlock prints a basic block.
func (p *Printer) PrintBlock(b *Block) {
	// Block header
	header := fmt.Sprintf("b%d:", b.ID)

	// Add predecessor info
	if len(b.Preds) > 0 {
		var preds []string
		for _, pred := range b.Preds {
			preds = append(preds, fmt.Sprintf("b%d", pred.ID))
		}
		header += fmt.Sprintf(" // preds: %s", strings.Join(preds, ", "))
	}

	p.writeLine("%s", header)

	// Print values
	for _, v := range b.Values {
		p.PrintValue(v)
	}

	// Print terminator
	p.printTerminator(b)

	p.writeLine("") // blank line after block
}

// PrintValue prints a single value.
func (p *Printer) PrintValue(v *Value) {
	p.writeLine("    %s", p.formatValue(v))
}

// formatValue returns the formatted string for a value.
func (p *Printer) formatValue(v *Value) string {
	var b strings.Builder

	// Result assignment
	b.WriteString(fmt.Sprintf("v%d = %s", v.ID, v.Op.String()))

	// Auxiliary data
	switch v.Op {
	case OpConst:
		if v.Type != nil {
			switch v.Type.(type) {
			case *StringType:
				b.WriteString(fmt.Sprintf(" %q", v.AuxString))
			case *BoolType:
				if v.AuxInt != 0 {
					b.WriteString(" true")
				} else {
					b.WriteString(" false")
				}
			default:
				b.WriteString(fmt.Sprintf(" %d", v.AuxInt))
			}
		} else {
			b.WriteString(fmt.Sprintf(" %d", v.AuxInt))
		}

	case OpArg:
		b.WriteString(fmt.Sprintf(" %%%d", v.AuxInt))

	case OpCall:
		b.WriteString(fmt.Sprintf(" @%s", v.AuxString))

	case OpFieldPtr:
		b.WriteString(fmt.Sprintf(" +%d", v.AuxInt))

	case OpAlloc:
		b.WriteString(fmt.Sprintf(" %d", v.AuxInt))

	case OpFree:
		b.WriteString(fmt.Sprintf(" size=%d", v.AuxInt))

	case OpPhi:
		b.WriteString(" [")
		for i, arg := range v.PhiArgs {
			if i > 0 {
				b.WriteString(", ")
			}
			if arg.Value != nil {
				b.WriteString(fmt.Sprintf("b%d: v%d", arg.From.ID, arg.Value.ID))
			} else {
				b.WriteString(fmt.Sprintf("b%d: nil", arg.From.ID))
			}
		}
		b.WriteString("]")
	}

	// Arguments (for non-phi operations)
	if len(v.Args) > 0 && v.Op != OpPhi {
		b.WriteString(" ")
		for i, arg := range v.Args {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("v%d", arg.ID))
		}
	}

	// Type annotation
	if v.Type != nil {
		b.WriteString(fmt.Sprintf(" : %s", v.Type.String()))
	}

	return b.String()
}

// printTerminator prints the block terminator.
func (p *Printer) printTerminator(b *Block) {
	switch b.Kind {
	case BlockPlain:
		if len(b.Succs) > 0 {
			p.writeLine("    jump -> b%d", b.Succs[0].ID)
		}

	case BlockIf:
		if b.Control != nil && len(b.Succs) >= 2 {
			p.writeLine("    if v%d -> b%d, b%d", b.Control.ID, b.Succs[0].ID, b.Succs[1].ID)
		}

	case BlockReturn:
		// Find the return value (last OpReturn in block)
		var retVal *Value
		for _, v := range b.Values {
			if v.Op == OpReturn {
				retVal = v
				break
			}
		}
		if retVal != nil && len(retVal.Args) > 0 {
			p.writeLine("    return v%d", retVal.Args[0].ID)
		} else {
			p.writeLine("    return")
		}

	case BlockExit:
		// Find the exit value
		var exitVal *Value
		for _, v := range b.Values {
			if v.Op == OpExit {
				exitVal = v
				break
			}
		}
		if exitVal != nil && len(exitVal.Args) > 0 {
			p.writeLine("    exit v%d", exitVal.Args[0].ID)
		} else {
			p.writeLine("    exit")
		}
	}
}

// writeLine writes an indented line to the output.
func (p *Printer) writeLine(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.w, line)
}

// String returns the IR as a string.
func String(prog *Program) string {
	var b strings.Builder
	NewPrinter(&b).PrintProgram(prog)
	return b.String()
}

// FunctionString returns a single function as a string.
func FunctionString(f *Function) string {
	var b strings.Builder
	NewPrinter(&b).PrintFunction(f)
	return b.String()
}

// BlockString returns a single block as a string.
func BlockString(b *Block) string {
	var sb strings.Builder
	NewPrinter(&sb).PrintBlock(b)
	return sb.String()
}
