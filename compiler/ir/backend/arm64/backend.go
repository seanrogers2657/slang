// Package arm64 provides an ARM64 code generation backend for the IR.
package arm64

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/compiler/ir"
	"github.com/seanrogers2657/slang/compiler/ir/backend"
)

// Backend generates ARM64 assembly from IR.
type Backend struct {
	config *backend.Config
}

// New creates a new ARM64 backend with the given configuration.
func New(config *backend.Config) *Backend {
	if config == nil {
		config = backend.DefaultConfig()
	}
	return &Backend{config: config}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "arm64"
}

// Generate produces ARM64 assembly from an IR program.
func (b *Backend) Generate(prog *ir.Program) (string, error) {
	g := newGenerator(prog, b.config)
	return g.generate()
}

// StackLayout holds the computed stack frame layout for a function.
// This is computed once and used during code generation.
type StackLayout struct {
	// Size is the total stack frame size (16-byte aligned).
	Size int
	// Offsets maps IR values to their stack slot offsets (relative to x29).
	Offsets map[*ir.Value]int
}

// LabelManager generates unique labels during code generation.
type LabelManager struct {
	counter     int
	blockLabels map[*ir.Block]string
}

// NewLabelManager creates a new label manager.
func NewLabelManager() *LabelManager {
	return &LabelManager{
		blockLabels: make(map[*ir.Block]string),
	}
}

// NextLabel returns a unique label number and increments the counter.
func (lm *LabelManager) NextLabel() int {
	label := lm.counter
	lm.counter++
	return label
}

// BlockLabel returns the label for a basic block.
func (lm *LabelManager) BlockLabel(block *ir.Block) string {
	return lm.blockLabels[block]
}

// SetBlockLabel sets the label for a basic block.
func (lm *LabelManager) SetBlockLabel(block *ir.Block, label string) {
	lm.blockLabels[block] = label
}

// InitBlockLabels initializes labels for all blocks in a function.
func (lm *LabelManager) InitBlockLabels(fn *ir.Function) {
	for _, block := range fn.Blocks {
		lm.blockLabels[block] = fmt.Sprintf("_%s_b%d", fn.Name, block.ID)
	}
}

// panicMessage defines a runtime panic with its assembly label and message.
type panicMessage struct {
	Label   string // Assembly label (e.g., "_sl_panic_div_zero")
	Message string // Error message including newline
}

// Len returns the length of the panic message.
func (p panicMessage) Len() int { return len(p.Message) }

// Runtime panic messages - message lengths are auto-computed.
var (
	PanicDivZero          = panicMessage{"_sl_panic_div_zero", "panic: division by zero\n"}
	PanicModZero          = panicMessage{"_sl_panic_mod_zero", "panic: modulo by zero\n"}
	PanicBounds           = panicMessage{"_sl_panic_bounds", "panic: array index out of bounds\n"}
	PanicOverflowAdd      = panicMessage{"_sl_panic_overflow_add", "panic: integer overflow: addition\n"}
	PanicOverflowSub      = panicMessage{"_sl_panic_overflow_sub", "panic: integer overflow: subtraction\n"}
	PanicOverflowMul      = panicMessage{"_sl_panic_overflow_mul", "panic: integer overflow: multiplication\n"}
	PanicUnsignedOverAdd  = panicMessage{"_sl_panic_unsigned_overflow_add", "panic: unsigned overflow: addition\n"}
	PanicUnsignedUnderSub = panicMessage{"_sl_panic_unsigned_underflow_sub", "panic: unsigned underflow: subtraction\n"}
	PanicUnsignedOverMul  = panicMessage{"_sl_panic_unsigned_overflow_mul", "panic: unsigned overflow: multiplication\n"}
)

// allPanicMessages lists all panic messages for data section emission.
var allPanicMessages = []panicMessage{
	PanicDivZero,
	PanicModZero,
	PanicBounds,
	PanicOverflowAdd,
	PanicOverflowSub,
	PanicOverflowMul,
	PanicUnsignedOverAdd,
	PanicUnsignedUnderSub,
	PanicUnsignedOverMul,
}

// generator holds state during code generation.
type generator struct {
	prog   *ir.Program
	config *backend.Config

	// Output
	builder strings.Builder

	// Label generation
	labels *LabelManager

	// Current function context
	fn     *ir.Function
	layout *StackLayout

	// String constants collected from the program
	strings []string
}

func newGenerator(prog *ir.Program, config *backend.Config) *generator {
	return &generator{
		prog:   prog,
		config: config,
		labels: NewLabelManager(),
	}
}

func (g *generator) generate() (string, error) {
	// Collect string constants from all functions
	g.collectStrings()

	// Emit data section if needed
	g.emitDataSection()

	// Emit text section
	g.emit(".text")
	g.emit(".align 4")
	g.emit("")

	// Emit _start entry point
	g.emitStartEntry()

	// Emit heap allocator
	g.emitHeapAllocator()

	// Emit print helper functions
	g.emitPrintHelpers()

	// Emit string comparison helper
	g.emitStrEqHelper()

	// Emit panic helper
	g.emitPanicHelper()

	// Generate each function
	for _, fn := range g.prog.Functions {
		if err := g.generateFunction(fn); err != nil {
			return "", err
		}
	}

	return g.builder.String(), nil
}

func (g *generator) collectStrings() {
	// Use strings from program (already collected by IR generator)
	g.strings = g.prog.Strings
}

func (g *generator) emitDataSection() {
	g.emit(".data")
	g.emit(".align 3")

	// Heap pointers (for dynamic allocator with free lists)
	g.emit("_sl_heap_ptr:")
	g.emit("    .quad 0")  // Current bump allocation pointer
	g.emit("_sl_heap_end:")
	g.emit("    .quad 0")  // End of current arena
	g.emit("_sl_arena_head:")
	g.emit("    .quad 0")  // Head of arena linked list
	g.emit("_sl_current_arena:")
	g.emit("    .quad 0")  // Current arena being bump-allocated from

	// Free list heads for size classes: 16, 32, 64, 128, 256, 512, 1024, 2048 bytes
	// Each entry is a pointer to the first free block of that size class (or 0 if empty)
	g.emit("_sl_free_lists:")
	g.emit("    .quad 0")  // Size class 0: 16 bytes
	g.emit("    .quad 0")  // Size class 1: 32 bytes
	g.emit("    .quad 0")  // Size class 2: 64 bytes
	g.emit("    .quad 0")  // Size class 3: 128 bytes
	g.emit("    .quad 0")  // Size class 4: 256 bytes
	g.emit("    .quad 0")  // Size class 5: 512 bytes
	g.emit("    .quad 0")  // Size class 6: 1024 bytes
	g.emit("    .quad 0")  // Size class 7: 2048 bytes

	// Newline for print
	g.emit("_sl_newline:")
	g.emitRaw(`    .asciz "\n"`)

	// Boolean strings
	g.emit("_sl_true_str:")
	g.emitRaw(`    .asciz "true"`)
	g.emit("_sl_false_str:")
	g.emitRaw(`    .asciz "false"`)

	// Assertion prefix
	g.emit("_sl_assert_prefix:")
	g.emitRaw(`    .asciz "assertion failed: "`)

	// Runtime error messages (from registry)
	for _, p := range allPanicMessages {
		g.emit("%s:", p.Label)
		g.emitRaw(`    .asciz "` + escapeString(p.Message) + `"`)
	}

	// Stack trace helper strings
	g.emit("_sl_panic_at_prefix:")
	g.emitRaw(`    .asciz "at "`)
	g.emit("_sl_panic_at_suffix:")
	g.emitRaw(`    .asciz "()\n"`)

	// Function name strings for stack traces
	for _, fn := range g.prog.Functions {
		g.emit("_sl_fn_name_%s:", fn.Name)
		g.emitRaw(`    .asciz "` + fn.Name + `"`)
	}

	// Emit string constants
	for i, s := range g.strings {
		g.emit("_sl_str%d:", i)
		g.emitRaw(`    .asciz "` + escapeString(s) + `"`)
	}

	// Heap is dynamically allocated via mmap at runtime

	// Global variables (from top-level var declarations)
	for _, global := range g.prog.Globals {
		g.emit("_sl_global_%s:", global.Name)
		g.emit("    .quad 0")
	}

	g.emit("")
}

// emitStartEntry emits the _start entry point
func (g *generator) emitStartEntry() {
	g.emit(".global _start")
	g.emit(".align 4")
	g.emit("_start:")
	g.emit("    // Initialize heap with first arena via mmap")
	g.emit("    bl _sl_heap_init")
	g.emit("    bl _main")
	// Default exit with code 0 if main returns
	g.emit("    mov x0, #0")
	g.emit("    mov x16, #1")
	g.emit("    svc #0")
	g.emit("")
}

// emitHeapAllocator emits a dynamic allocator with free lists and bump allocation
func (g *generator) emitHeapAllocator() {
	// Size classes: 16, 32, 64, 128, 256, 512, 1024, 2048 bytes
	// Sizes > 2048 use bump allocation only (no free list)

	// Heap initialization - allocate first arena via mmap
	// Arena header layout (32 bytes):
	//   offset 0:  next_arena pointer (8 bytes)
	//   offset 8:  alloc_count (8 bytes) - number of live allocations
	//   offset 16: arena_start (8 bytes) - start of this arena (for munmap)
	//   offset 24: arena_size (8 bytes) - size of this arena (for munmap)
	g.emit("// Heap initializer - allocate first arena")
	g.emit("_sl_heap_init:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    mov x0, #0")                              // addr = NULL (let kernel choose)
	g.emit("    movz x1, #0x10, lsl #16")                 // len = 1MB (0x100000 = 16 << 16)
	g.emit("    mov x2, #3")                              // prot = PROT_READ | PROT_WRITE
	g.emit("    mov x3, #0x1002")                         // flags = MAP_PRIVATE | MAP_ANON
	g.emit("    mov x4, #0")
	g.emit("    sub x4, x4, #1")                          // fd = -1 (anonymous)
	g.emit("    mov x5, #0")                              // offset = 0
	g.emit("    mov x16, #197")                           // mmap syscall
	g.emit("    svc #0")
	g.emit("    adds xzr, x0, #1")                        // Check for MAP_FAILED (-1)
	g.emit("    b.eq _sl_heap_init_fail")
	g.emit("    // Initialize arena header")
	g.emit("    str xzr, [x0]")                           // next_arena = NULL
	g.emit("    str xzr, [x0, #8]")                       // alloc_count = 0
	g.emit("    str x0, [x0, #16]")                       // arena_start = x0
	g.emit("    movz x10, #0x10, lsl #16")                // arena_size = 1MB
	g.emit("    str x10, [x0, #24]")
	g.emit("    // Store heap_ptr (skip 32-byte header)")
	g.emit("    add x11, x0, #32")                        // heap_ptr starts after header
	g.emit("    adrp x9, _sl_heap_ptr@PAGE")
	g.emit("    add x9, x9, _sl_heap_ptr@PAGEOFF")
	g.emit("    str x11, [x9]")
	g.emit("    // Store heap_end")
	g.emit("    adrp x9, _sl_heap_end@PAGE")
	g.emit("    add x9, x9, _sl_heap_end@PAGEOFF")
	g.emit("    add x10, x0, x10")                        // x10 = arena_start + 1MB
	g.emit("    str x10, [x9]")
	g.emit("    // Store arena_head and current_arena")
	g.emit("    adrp x9, _sl_arena_head@PAGE")
	g.emit("    add x9, x9, _sl_arena_head@PAGEOFF")
	g.emit("    str x0, [x9]")
	g.emit("    adrp x9, _sl_current_arena@PAGE")
	g.emit("    add x9, x9, _sl_current_arena@PAGEOFF")
	g.emit("    str x0, [x9]")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("_sl_heap_init_fail:")
	g.emit("    mov x0, #1")
	g.emit("    mov x16, #1")
	g.emit("    svc #0")
	g.emit("")

	// Get size class index from size (x0 = size, returns x1 = class index 0-7, or 8 if too large)
	// Size classes: 16, 32, 64, 128, 256, 512, 1024, 2048
	g.emit("// Get size class: x0 = size, returns x1 = class index (0-7, or 8 if >2048)")
	g.emit("_sl_size_class:")
	g.emit("    cmp x0, #16")
	g.emit("    b.le _sl_sc_0")
	g.emit("    cmp x0, #32")
	g.emit("    b.le _sl_sc_1")
	g.emit("    cmp x0, #64")
	g.emit("    b.le _sl_sc_2")
	g.emit("    cmp x0, #128")
	g.emit("    b.le _sl_sc_3")
	g.emit("    cmp x0, #256")
	g.emit("    b.le _sl_sc_4")
	g.emit("    cmp x0, #512")
	g.emit("    b.le _sl_sc_5")
	g.emit("    cmp x0, #1024")
	g.emit("    b.le _sl_sc_6")
	g.emit("    cmp x0, #2048")
	g.emit("    b.le _sl_sc_7")
	g.emit("    mov x1, #8")           // Too large for free list
	g.emit("    ret")
	g.emit("_sl_sc_0:")
	g.emit("    mov x1, #0")
	g.emit("    mov x0, #16")          // Round up to size class
	g.emit("    ret")
	g.emit("_sl_sc_1:")
	g.emit("    mov x1, #1")
	g.emit("    mov x0, #32")
	g.emit("    ret")
	g.emit("_sl_sc_2:")
	g.emit("    mov x1, #2")
	g.emit("    mov x0, #64")
	g.emit("    ret")
	g.emit("_sl_sc_3:")
	g.emit("    mov x1, #3")
	g.emit("    mov x0, #128")
	g.emit("    ret")
	g.emit("_sl_sc_4:")
	g.emit("    mov x1, #4")
	g.emit("    mov x0, #256")
	g.emit("    ret")
	g.emit("_sl_sc_5:")
	g.emit("    mov x1, #5")
	g.emit("    mov x0, #512")
	g.emit("    ret")
	g.emit("_sl_sc_6:")
	g.emit("    mov x1, #6")
	g.emit("    mov x0, #1024")
	g.emit("    ret")
	g.emit("_sl_sc_7:")
	g.emit("    mov x1, #7")
	g.emit("    mov x0, #2048")
	g.emit("    ret")
	g.emit("")

	// Heap grow - allocate a new arena when current one is exhausted
	g.emit("// Heap grow - allocate new arena")
	g.emit("_sl_heap_grow:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    stp x19, x20, [sp, #-16]!")
	g.emit("    mov x19, x0")                             // Save requested size
	g.emit("    // Allocate new arena")
	g.emit("    mov x0, #0")
	g.emit("    movz x1, #0x10, lsl #16")                 // len = 1MB
	g.emit("    mov x2, #3")
	g.emit("    mov x3, #0x1002")
	g.emit("    mov x4, #0")
	g.emit("    sub x4, x4, #1")
	g.emit("    mov x5, #0")
	g.emit("    mov x16, #197")
	g.emit("    svc #0")
	g.emit("    adds xzr, x0, #1")
	g.emit("    b.eq _sl_heap_grow_fail")
	g.emit("    mov x20, x0")                             // x20 = new arena start
	g.emit("    // Link new arena to head of list")
	g.emit("    adrp x9, _sl_arena_head@PAGE")
	g.emit("    add x9, x9, _sl_arena_head@PAGEOFF")
	g.emit("    ldr x10, [x9]")                           // x10 = old head
	g.emit("    str x10, [x20]")                          // new_arena->next = old_head
	g.emit("    str x20, [x9]")                           // arena_head = new_arena
	g.emit("    // Initialize arena header")
	g.emit("    str xzr, [x20, #8]")                      // alloc_count = 0
	g.emit("    str x20, [x20, #16]")                     // arena_start
	g.emit("    movz x10, #0x10, lsl #16")
	g.emit("    str x10, [x20, #24]")                     // arena_size = 1MB
	g.emit("    // Update heap_ptr and heap_end")
	g.emit("    add x11, x20, #32")                       // heap_ptr after header
	g.emit("    adrp x9, _sl_heap_ptr@PAGE")
	g.emit("    add x9, x9, _sl_heap_ptr@PAGEOFF")
	g.emit("    str x11, [x9]")
	g.emit("    adrp x9, _sl_heap_end@PAGE")
	g.emit("    add x9, x9, _sl_heap_end@PAGEOFF")
	g.emit("    add x10, x20, x10")
	g.emit("    str x10, [x9]")
	g.emit("    // Update current_arena")
	g.emit("    adrp x9, _sl_current_arena@PAGE")
	g.emit("    add x9, x9, _sl_current_arena@PAGEOFF")
	g.emit("    str x20, [x9]")
	g.emit("    // Retry bump allocation")
	g.emit("    mov x0, x19")
	g.emit("    ldp x19, x20, [sp], #16")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    b _sl_bump_alloc")
	g.emit("_sl_heap_grow_fail:")
	g.emit("    mov x0, #1")
	g.emit("    mov x16, #1")
	g.emit("    svc #0")
	g.emit("")

	// Bump allocator (internal) - x0 = size (already rounded to size class)
	g.emit("// Bump allocator: x0 = size")
	g.emit("_sl_bump_alloc:")
	g.emit("    adrp x9, _sl_heap_ptr@PAGE")
	g.emit("    add x9, x9, _sl_heap_ptr@PAGEOFF")
	g.emit("    ldr x10, [x9]")                           // x10 = heap_ptr
	g.emit("    add x11, x10, x0")                        // x11 = heap_ptr + size
	g.emit("    adrp x12, _sl_heap_end@PAGE")
	g.emit("    add x12, x12, _sl_heap_end@PAGEOFF")
	g.emit("    ldr x13, [x12]")                          // x13 = heap_end
	g.emit("    cmp x11, x13")
	g.emit("    b.gt _sl_heap_grow")                      // If new_ptr > end, grow heap
	g.emit("    str x11, [x9]")                           // heap_ptr = new_ptr
	g.emit("    // Increment current arena's alloc_count")
	g.emit("    adrp x12, _sl_current_arena@PAGE")
	g.emit("    add x12, x12, _sl_current_arena@PAGEOFF")
	g.emit("    ldr x13, [x12]")                          // x13 = current_arena
	g.emit("    ldr x14, [x13, #8]")                      // x14 = alloc_count
	g.emit("    add x14, x14, #1")
	g.emit("    str x14, [x13, #8]")                      // alloc_count++
	g.emit("    mov x0, x10")                             // Return old heap_ptr
	g.emit("    ret")
	g.emit("")

	// Main allocation function with free list support
	g.emit("// Allocator with free list: x0 = size to allocate")
	g.emit("_sl_heap_alloc:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    // Get size class")
	g.emit("    bl _sl_size_class")                       // x1 = class index, x0 = rounded size
	g.emit("    cmp x1, #8")
	g.emit("    b.eq _sl_alloc_bump")                     // Size > 2048, use bump allocator
	g.emit("    // Check free list for this size class")
	g.emit("    adrp x9, _sl_free_lists@PAGE")
	g.emit("    add x9, x9, _sl_free_lists@PAGEOFF")
	g.emit("    lsl x10, x1, #3")                         // x10 = class * 8 (offset into free_lists)
	g.emit("    add x9, x9, x10")                         // x9 = &free_lists[class]
	g.emit("    ldr x11, [x9]")                           // x11 = free_lists[class] (head pointer)
	g.emit("    cbz x11, _sl_alloc_bump")                 // If empty, use bump allocator
	g.emit("    // Pop from free list")
	g.emit("    ldr x12, [x11]")                          // x12 = next pointer (stored in freed block)
	g.emit("    str x12, [x9]")                           // free_lists[class] = next
	g.emit("    // Save block pointer (x11 will be clobbered by find_arena)")
	g.emit("    stp x11, xzr, [sp, #-16]!")               // Push block pointer
	g.emit("    // Find arena for this block and increment alloc_count")
	g.emit("    mov x0, x11")                             // x0 = block pointer
	g.emit("    bl _sl_find_arena")                       // x0 = arena or 0
	g.emit("    cbz x0, _sl_alloc_freelist_done")         // Arena not found (shouldn't happen)
	g.emit("    ldr x12, [x0, #8]")                       // x12 = alloc_count
	g.emit("    add x12, x12, #1")
	g.emit("    str x12, [x0, #8]")                       // alloc_count++
	g.emit("_sl_alloc_freelist_done:")
	g.emit("    ldp x11, xzr, [sp], #16")                 // Restore block pointer
	g.emit("    mov x0, x11")                             // Return the freed block
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("_sl_alloc_bump:")
	g.emit("    // Use bump allocator")
	g.emit("    bl _sl_bump_alloc")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")

	// Find arena for a pointer - walks arena list
	g.emit("// Find arena containing pointer: x0 = ptr, returns x0 = arena or 0")
	g.emit("_sl_find_arena:")
	g.emit("    adrp x9, _sl_arena_head@PAGE")
	g.emit("    add x9, x9, _sl_arena_head@PAGEOFF")
	g.emit("    ldr x10, [x9]")                           // x10 = arena_head
	g.emit("_sl_find_arena_loop:")
	g.emit("    cbz x10, _sl_find_arena_not_found")       // End of list
	g.emit("    ldr x11, [x10, #16]")                     // x11 = arena_start
	g.emit("    ldr x12, [x10, #24]")                     // x12 = arena_size
	g.emit("    add x12, x11, x12")                       // x12 = arena_end
	g.emit("    cmp x0, x11")
	g.emit("    b.lt _sl_find_arena_next")                // ptr < arena_start
	g.emit("    cmp x0, x12")
	g.emit("    b.ge _sl_find_arena_next")                // ptr >= arena_end
	g.emit("    // Found: ptr is in [arena_start, arena_end)")
	g.emit("    mov x0, x10")
	g.emit("    ret")
	g.emit("_sl_find_arena_next:")
	g.emit("    ldr x10, [x10]")                          // x10 = arena->next
	g.emit("    b _sl_find_arena_loop")
	g.emit("_sl_find_arena_not_found:")
	g.emit("    mov x0, #0")
	g.emit("    ret")
	g.emit("")

	// Free function - adds block to free list and decrements arena alloc_count
	g.emit("// Free: x0 = pointer, x1 = size")
	g.emit("_sl_heap_free:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    stp x19, x20, [sp, #-16]!")
	g.emit("    stp x21, x22, [sp, #-16]!")
	g.emit("    mov x19, x0")                             // x19 = pointer to free
	g.emit("    mov x20, x1")                             // x20 = size
	g.emit("    // Find arena for this pointer")
	g.emit("    bl _sl_find_arena")                       // x0 = arena
	g.emit("    mov x21, x0")                             // x21 = arena
	g.emit("    cbz x21, _sl_free_no_arena")              // Arena not found (shouldn't happen)
	g.emit("    // Decrement alloc_count")
	g.emit("    ldr x22, [x21, #8]")                      // x22 = alloc_count
	g.emit("    sub x22, x22, #1")
	g.emit("    str x22, [x21, #8]")                      // alloc_count--
	g.emit("    cbnz x22, _sl_free_add_to_list")          // If count > 0, add to free list
	g.emit("    // Arena is empty - check if it's the only arena")
	g.emit("    ldr x9, [x21]")                           // x9 = arena->next
	g.emit("    cbnz x9, _sl_free_unmap")                 // If has next, safe to unmap
	g.emit("    adrp x9, _sl_arena_head@PAGE")
	g.emit("    add x9, x9, _sl_arena_head@PAGEOFF")
	g.emit("    ldr x9, [x9]")                            // x9 = arena_head
	g.emit("    cmp x9, x21")
	g.emit("    b.eq _sl_free_add_to_list")               // If this is head and only, keep it
	g.emit("_sl_free_unmap:")
	g.emit("    // Arena is empty - unmap it")
	g.emit("    mov x0, x21")
	g.emit("    bl _sl_unmap_arena")
	g.emit("    b _sl_free_done")
	g.emit("_sl_free_no_arena:")
	g.emit("_sl_free_add_to_list:")
	g.emit("    // Add to free list")
	g.emit("    mov x0, x20")                             // x0 = size
	g.emit("    bl _sl_size_class")                       // x1 = class index
	g.emit("    cmp x1, #8")
	g.emit("    b.eq _sl_free_done")                      // Size > 2048, can't add to free list
	g.emit("    // Push onto free list")
	g.emit("    adrp x9, _sl_free_lists@PAGE")
	g.emit("    add x9, x9, _sl_free_lists@PAGEOFF")
	g.emit("    lsl x10, x1, #3")                         // x10 = class * 8
	g.emit("    add x9, x9, x10")                         // x9 = &free_lists[class]
	g.emit("    ldr x11, [x9]")                           // x11 = current head
	g.emit("    str x11, [x19]")                          // Store old head in freed block
	g.emit("    str x19, [x9]")                           // free_lists[class] = freed block
	g.emit("_sl_free_done:")
	g.emit("    ldp x21, x22, [sp], #16")
	g.emit("    ldp x19, x20, [sp], #16")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")

	// Unmap arena - removes from list, cleans free lists, and calls munmap
	g.emit("// Unmap arena: x0 = arena to unmap")
	g.emit("_sl_unmap_arena:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    stp x19, x20, [sp, #-16]!")
	g.emit("    stp x21, x22, [sp, #-16]!")
	g.emit("    mov x19, x0")                             // x19 = arena to unmap
	g.emit("    // Remove arena from linked list")
	g.emit("    adrp x9, _sl_arena_head@PAGE")
	g.emit("    add x9, x9, _sl_arena_head@PAGEOFF")
	g.emit("    ldr x10, [x9]")                           // x10 = arena_head
	g.emit("    cmp x10, x19")
	g.emit("    b.ne _sl_unmap_find_prev")
	g.emit("    // Arena is head - update head to next")
	g.emit("    ldr x11, [x19]")                          // x11 = arena->next")
	g.emit("    str x11, [x9]")                           // arena_head = arena->next
	g.emit("    b _sl_unmap_clean_freelists")
	g.emit("_sl_unmap_find_prev:")
	g.emit("    // Find previous arena in list")
	g.emit("    mov x20, x10")                            // x20 = prev
	g.emit("_sl_unmap_find_prev_loop:")
	g.emit("    ldr x10, [x20]")                          // x10 = prev->next
	g.emit("    cbz x10, _sl_unmap_clean_freelists")      // End of list (shouldn't happen)")
	g.emit("    cmp x10, x19")
	g.emit("    b.eq _sl_unmap_found_prev")
	g.emit("    mov x20, x10")
	g.emit("    b _sl_unmap_find_prev_loop")
	g.emit("_sl_unmap_found_prev:")
	g.emit("    // x20 = prev, x19 = arena to remove")
	g.emit("    ldr x11, [x19]")                          // x11 = arena->next
	g.emit("    str x11, [x20]")                          // prev->next = arena->next
	g.emit("_sl_unmap_clean_freelists:")
	g.emit("    // Remove any free list entries pointing into this arena")
	g.emit("    ldr x20, [x19, #16]")                     // x20 = arena_start
	g.emit("    ldr x21, [x19, #24]")                     // x21 = arena_size
	g.emit("    add x21, x20, x21")                       // x21 = arena_end
	g.emit("    mov x22, #0")                             // x22 = size class index
	g.emit("_sl_unmap_clean_class_loop:")
	g.emit("    cmp x22, #8")
	g.emit("    b.ge _sl_unmap_do_munmap")
	g.emit("    adrp x9, _sl_free_lists@PAGE")
	g.emit("    add x9, x9, _sl_free_lists@PAGEOFF")
	g.emit("    lsl x10, x22, #3")
	g.emit("    add x9, x9, x10")                         // x9 = &free_lists[class]
	g.emit("    // Remove entries from this class that are in the arena")
	g.emit("    bl _sl_clean_freelist_for_arena")         // x9=list head ptr, x20=start, x21=end
	g.emit("    add x22, x22, #1")
	g.emit("    b _sl_unmap_clean_class_loop")
	g.emit("_sl_unmap_do_munmap:")
	g.emit("    // If this was current_arena, we need to update current_arena")
	g.emit("    adrp x9, _sl_current_arena@PAGE")
	g.emit("    add x9, x9, _sl_current_arena@PAGEOFF")
	g.emit("    ldr x10, [x9]")
	g.emit("    cmp x10, x19")
	g.emit("    b.ne _sl_unmap_call_munmap")
	g.emit("    // Current arena is being unmapped - set to head")
	g.emit("    adrp x10, _sl_arena_head@PAGE")
	g.emit("    add x10, x10, _sl_arena_head@PAGEOFF")
	g.emit("    ldr x10, [x10]")
	g.emit("    str x10, [x9]")                           // current_arena = arena_head
	g.emit("_sl_unmap_call_munmap:")
	g.emit("    // Call munmap(arena_start, arena_size)")
	g.emit("    ldr x0, [x19, #16]")                      // x0 = arena_start
	g.emit("    ldr x1, [x19, #24]")                      // x1 = arena_size
	g.emit("    mov x16, #73")                            // munmap syscall
	g.emit("    svc #0")
	g.emit("    ldp x21, x22, [sp], #16")
	g.emit("    ldp x19, x20, [sp], #16")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")

	// Clean free list entries for a specific arena
	// x9 = pointer to free list head, x20 = arena_start, x21 = arena_end
	g.emit("// Clean freelist entries in arena range")
	g.emit("_sl_clean_freelist_for_arena:")
	g.emit("    // Remove entries where arena_start <= entry < arena_end")
	g.emit("    // We need to walk the list and skip any entries in range")
	g.emit("    mov x10, x9")                             // x10 = pointer to current (starts at head ptr)
	g.emit("_sl_clean_freelist_loop:")
	g.emit("    ldr x11, [x10]")                          // x11 = current entry
	g.emit("    cbz x11, _sl_clean_freelist_done")        // End of list
	g.emit("    cmp x11, x20")
	g.emit("    b.lt _sl_clean_freelist_keep")            // entry < arena_start, keep it
	g.emit("    cmp x11, x21")
	g.emit("    b.ge _sl_clean_freelist_keep")            // entry >= arena_end, keep it
	g.emit("    // Entry is in arena range - skip it")
	g.emit("    ldr x12, [x11]")                          // x12 = entry->next
	g.emit("    str x12, [x10]")                          // prev->next = entry->next (or head = entry->next)
	g.emit("    b _sl_clean_freelist_loop")               // Don't advance x10, check new entry at same position
	g.emit("_sl_clean_freelist_keep:")
	g.emit("    mov x10, x11")                            // prev = current")
	g.emit("    b _sl_clean_freelist_loop")
	g.emit("_sl_clean_freelist_done:")
	g.emit("    ret")
	g.emit("")
}

// emitPrintHelpers emits helper functions for printing
func (g *generator) emitPrintHelpers() {
	// Print integer helper - converts int to string and prints
	g.emit("// Print integer (x0 = value)")
	g.emit("_sl_print_int:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    sub sp, sp, #32")         // Buffer for digits
	g.emit("    mov x10, x0")             // Value to print
	g.emit("    mov x11, sp")             // Buffer pointer (end)
	g.emit("    add x11, x11, #30")
	g.emit("    mov x12, #0")             // Digit count
	g.emit("    mov x13, #0")             // Negative flag

	// Handle negative
	g.emit("    cmp x10, #0")
	g.emit("    bge _sl_print_int_positive")
	g.emit("    mov x13, #1")
	g.emit("    neg x10, x10")

	g.emit("_sl_print_int_positive:")
	g.emit("    mov x14, #10")

	g.emit("_sl_print_int_loop:")
	g.emit("    udiv x15, x10, x14")      // x15 = x10 / 10
	g.emit("    msub x16, x15, x14, x10") // x16 = x10 % 10
	g.emit("    add x16, x16, #48")       // Convert to ASCII
	g.emit("    strb w16, [x11]")
	g.emit("    sub x11, x11, #1")
	g.emit("    add x12, x12, #1")
	g.emit("    mov x10, x15")
	g.emit("    cbnz x10, _sl_print_int_loop")

	// Add minus sign if negative
	g.emit("    cbz x13, _sl_print_int_write")
	g.emit("    mov x16, #45")            // '-'
	g.emit("    strb w16, [x11]")
	g.emit("    sub x11, x11, #1")
	g.emit("    add x12, x12, #1")

	g.emit("_sl_print_int_write:")
	g.emit("    add x11, x11, #1")        // Point to first digit
	g.emit("    mov x0, #1")              // stdout
	g.emit("    mov x1, x11")             // buffer
	g.emit("    mov x2, x12")             // length
	g.emit("    mov x16, #4")             // write syscall
	g.emit("    svc #0")

	// Print newline
	g.emit("    adrp x1, _sl_newline@PAGE")
	g.emit("    add x1, x1, _sl_newline@PAGEOFF")
	g.emit("    mov x0, #1")
	g.emit("    mov x2, #1")
	g.emit("    mov x16, #4")
	g.emit("    svc #0")

	g.emit("    add sp, sp, #32")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")

	// Print string helper
	g.emit("// Print string (x0 = pointer)")
	g.emit("_sl_print_str:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    mov x10, x0")             // String pointer

	// Calculate string length
	g.emit("    mov x11, x0")
	g.emit("_sl_strlen_loop:")
	g.emit("    ldrb w12, [x11]")
	g.emit("    cbz w12, _sl_strlen_done")
	g.emit("    add x11, x11, #1")
	g.emit("    b _sl_strlen_loop")
	g.emit("_sl_strlen_done:")
	g.emit("    sub x2, x11, x10")        // Length

	g.emit("    mov x0, #1")              // stdout
	g.emit("    mov x1, x10")             // buffer
	g.emit("    mov x16, #4")             // write syscall
	g.emit("    svc #0")

	// Print newline
	g.emit("    adrp x1, _sl_newline@PAGE")
	g.emit("    add x1, x1, _sl_newline@PAGEOFF")
	g.emit("    mov x0, #1")
	g.emit("    mov x2, #1")
	g.emit("    mov x16, #4")
	g.emit("    svc #0")

	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")

	// Print bool helper
	g.emit("// Print bool (x0 = 0 or 1)")
	g.emit("_sl_print_bool:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    cbz x0, _sl_print_false")

	g.emit("    adrp x0, _sl_true_str@PAGE")
	g.emit("    add x0, x0, _sl_true_str@PAGEOFF")
	g.emit("    bl _sl_print_str")
	g.emit("    b _sl_print_bool_done")

	g.emit("_sl_print_false:")
	g.emit("    adrp x0, _sl_false_str@PAGE")
	g.emit("    add x0, x0, _sl_false_str@PAGEOFF")
	g.emit("    bl _sl_print_str")

	g.emit("_sl_print_bool_done:")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")
}

// emitPanicHelper emits a helper to print an error message to stderr and exit
// emitStrEqHelper emits a helper that compares two null-terminated strings.
// x0 = pointer to string A, x1 = pointer to string B
// Returns x0 = 1 if equal, 0 if not.
func (g *generator) emitStrEqHelper() {
	g.emit("// String equality helper")
	g.emit("_sl_str_eq:")
	g.emit("    mov x10, x0")               // copy A ptr
	g.emit("    mov x11, x1")               // copy B ptr
	g.emit("_sl_str_eq_loop:")
	g.emit("    ldrb w12, [x10]")            // load byte from A
	g.emit("    ldrb w14, [x11]")            // load byte from B
	g.emit("    cmp x12, x14")              // compare as 64-bit (upper bits are zero)
	g.emit("    b.ne _sl_str_eq_false")      // bytes differ
	g.emit("    cbz x12, _sl_str_eq_true")   // both null terminator
	g.emit("    add x10, x10, #1")
	g.emit("    add x11, x11, #1")
	g.emit("    b _sl_str_eq_loop")
	g.emit("_sl_str_eq_true:")
	g.emit("    mov x0, #1")
	g.emit("    ret")
	g.emit("_sl_str_eq_false:")
	g.emit("    mov x0, #0")
	g.emit("    ret")
	g.emit("")
}

func (g *generator) emitPanicHelper() {
	// _sl_panic: x0 = error message ptr, x1 = error msg len, x2 = func name ptr, x3 = func name len
	g.emit("// Panic helper - prints error to stderr and exits with code 1")
	g.emit("_sl_panic:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")

	// Save function name for later
	g.emit("    mov x19, x2")        // Save func name pointer
	g.emit("    mov x20, x3")        // Save func name length

	// Print error message
	g.emit("    mov x2, x1")         // length
	g.emit("    mov x1, x0")         // message
	g.emit("    mov x0, #2")         // stderr
	g.emit("    mov x16, #4")        // write syscall
	g.emit("    svc #0")

	// Print "at "
	g.emit("    adrp x1, _sl_panic_at_prefix@PAGE")
	g.emit("    add x1, x1, _sl_panic_at_prefix@PAGEOFF")
	g.emit("    mov x0, #2")         // stderr
	g.emit("    mov x2, #3")         // length of "at "
	g.emit("    mov x16, #4")
	g.emit("    svc #0")

	// Print function name
	g.emit("    mov x1, x19")        // func name
	g.emit("    mov x2, x20")        // func name length
	g.emit("    mov x0, #2")         // stderr
	g.emit("    mov x16, #4")
	g.emit("    svc #0")

	// Print "()\n"
	g.emit("    adrp x1, _sl_panic_at_suffix@PAGE")
	g.emit("    add x1, x1, _sl_panic_at_suffix@PAGEOFF")
	g.emit("    mov x0, #2")         // stderr
	g.emit("    mov x2, #3")         // length of "()\n"
	g.emit("    mov x16, #4")
	g.emit("    svc #0")

	// Exit with code 1
	g.emit("    mov x0, #1")
	g.emit("    mov x16, #1")
	g.emit("    svc #0")
	g.emit("")
}

func (g *generator) generateFunction(fn *ir.Function) error {
	g.fn = fn
	g.layout = ComputeStackLayout(fn)
	g.labels.InitBlockLabels(fn)

	g.emitPrologue()

	if err := g.generateBody(); err != nil {
		return err
	}

	return nil
}

// emitPrologue emits the function header, frame setup, and parameter stores.
func (g *generator) emitPrologue() {
	// Emit function header
	if g.fn.Name == "main" {
		g.emit(".global _main")
		g.emit("_main:")
	} else {
		g.emit("_%s:", g.fn.Name)
	}

	// Save frame pointer and link register
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")

	// Allocate stack space for locals
	if g.layout.Size > 0 {
		g.emit("    sub sp, sp, #%d", g.layout.Size)
	}

	// Store parameters to stack. Nullable value-type params use two registers (tag + value).
	regIdx := 0
	for _, param := range g.fn.Params {
		offset := g.stackOffset(param)
		if nullType, ok := param.Type.(*ir.NullableType); ok && !nullType.IsReferenceNullable() {
			// Nullable value type: tag in regIdx, value in regIdx+1
			g.emit("    str x%d, [x29, #%d]", regIdx, offset)   // tag
			g.emit("    str x%d, [x29, #%d]", regIdx+1, offset+8) // value
			regIdx += 2
		} else if regIdx < 8 {
			g.emit("    str x%d, [x29, #%d]", regIdx, offset)
			regIdx++
		} else {
			callerOffset := 16 + (regIdx-8)*8
			g.emit("    ldr x9, [x29, #%d]", callerOffset)
			g.storeToStack("x9", offset)
			regIdx++
		}
	}
}

// generateBody generates code for all blocks in the current function.
func (g *generator) generateBody() error {
	for _, block := range g.fn.Blocks {
		if err := g.generateBlock(block); err != nil {
			return err
		}
	}
	return nil
}

// emitReturnValue loads the return value into the appropriate registers.
func (g *generator) emitReturnValue(block *ir.Block) {
	// Find return value
	var retVal *ir.Value
	for _, v := range block.Values {
		if v.Op == ir.OpReturn {
			retVal = v
			break
		}
	}

	if retVal == nil || len(retVal.Args) == 0 {
		return
	}

	arg := retVal.Args[0]
	// Check if returning a value-type nullable (needs x0 + x1)
	if nullType, ok := arg.Type.(*ir.NullableType); ok && !nullType.IsReferenceNullable() {
		// Load tag into x0, value into x1
		if argOffset, ok := g.layout.Offsets[arg]; ok {
			g.loadFromStack("x0", argOffset)   // tag
			g.loadFromStack("x1", argOffset+8) // value
		} else {
			g.loadValue(arg, "x0")
			g.emit("    mov x1, #0")
		}
	} else {
		g.loadValue(arg, "x0")
	}
}

// emitEpilogue emits the function exit sequence: restore stack and return.
func (g *generator) emitEpilogue() {
	if g.layout.Size > 0 {
		g.emit("    add sp, sp, #%d", g.layout.Size)
	}
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
}

// ComputeStackLayout calculates the stack frame layout for a function.
// This is a pure function that returns a StackLayout without side effects.
func ComputeStackLayout(fn *ir.Function) *StackLayout {
	offsets := make(map[*ir.Value]int)
	offset := -16 // Start below saved x29, x30

	// Allocate space for parameters
	for _, param := range fn.Params {
		size := 8
		if nullType, ok := param.Type.(*ir.NullableType); ok && !nullType.IsReferenceNullable() {
			size = 16 // tag + value
		}
		offset -= size
		offsets[param] = offset
	}

	// Allocate space for all values that need stack storage
	for _, block := range fn.Blocks {
		for _, v := range block.Values {
			if needsStackSlot(v) {
				offset -= valueSize(v)
				offsets[v] = offset
			}
		}
	}

	// Calculate size: offset is negative, we need -offset bytes below x29
	// offset starts at -16 and decreases, so the furthest access is at 'offset'
	// We need sp <= x29 + offset, so stackSize = -offset
	size := -offset
	// Align to 16 bytes
	if size%16 != 0 {
		size += 16 - (size % 16)
	}

	return &StackLayout{
		Size:    size,
		Offsets: offsets,
	}
}

// stackOffset returns the stack offset for a value.
func (g *generator) stackOffset(v *ir.Value) int {
	return g.layout.Offsets[v]
}

// needsStackSlot determines if a value needs stack storage.
func needsStackSlot(v *ir.Value) bool {
	// Most values need stack slots in this simple register allocator
	switch v.Op {
	case ir.OpStore, ir.OpStoreGlobal, ir.OpFree, ir.OpReturn, ir.OpExit:
		return false // These don't produce values
	case ir.OpConst:
		return false // Constants are materialized inline, no stack needed
	case ir.OpPhi:
		return true // Phi nodes need slots for their merged values
	default:
		return v.Type != nil && !v.Type.Equal(ir.TypeVoid)
	}
}

// valueSize returns the stack size needed for a value.
func valueSize(v *ir.Value) int {
	if v.Type == nil {
		return 8
	}
	// Minimum stack slot is 8 bytes
	return max(v.Type.Size(), 8)
}

func (g *generator) generateBlock(block *ir.Block) error {
	// Emit block label (skip for entry block unless it has predecessors)
	if block.ID > 0 || len(block.Preds) > 0 {
		g.emit("%s:", g.labels.BlockLabel(block))
	}

	// Generate phi node copies from predecessors (handled at control flow)
	// Phi nodes are resolved when jumping to this block

	// Generate code for each value
	for _, v := range block.Values {
		if err := g.generateValue(v); err != nil {
			return err
		}
	}

	// Generate terminator
	return g.generateTerminator(block)
}

func (g *generator) generateValue(v *ir.Value) error {
	switch v.Op {
	case ir.OpConst:
		return g.genConst(v)
	case ir.OpArg:
		// Args are already stored in stack by prologue
		return nil
	case ir.OpAdd:
		return g.genAdd(v)
	case ir.OpSub:
		return g.genSub(v)
	case ir.OpMul:
		return g.genMul(v)
	case ir.OpDiv:
		return g.genDiv(v)
	case ir.OpMod:
		return g.genMod(v)
	case ir.OpNeg:
		return g.genNeg(v)
	case ir.OpEq:
		return g.genCmp(v, "eq")
	case ir.OpNe:
		return g.genCmp(v, "ne")
	case ir.OpLt:
		return g.genCmp(v, "lt")
	case ir.OpLe:
		return g.genCmp(v, "le")
	case ir.OpGt:
		return g.genCmp(v, "gt")
	case ir.OpGe:
		return g.genCmp(v, "ge")
	case ir.OpStrEq:
		return g.genStrEq(v)
	case ir.OpAnd:
		return g.genBinaryOp(v, "and")
	case ir.OpOr:
		return g.genBinaryOp(v, "orr")
	case ir.OpNot:
		return g.genNot(v)
	case ir.OpAlloc:
		return g.genAlloc(v)
	case ir.OpLoad:
		return g.genLoad(v)
	case ir.OpLoadGlobal:
		return g.genLoadGlobal(v)
	case ir.OpStoreGlobal:
		return g.genStoreGlobal(v)
	case ir.OpStore:
		return g.genStore(v)
	case ir.OpFree:
		return g.genFree(v)
	case ir.OpMemCopy:
		return g.genMemCopy(v)
	case ir.OpFieldPtr:
		return g.genFieldPtr(v)
	case ir.OpIndexPtr:
		return g.genIndexPtr(v)
	case ir.OpArrayLen:
		return g.genArrayLen(v)
	case ir.OpIsNull:
		return g.genIsNull(v)
	case ir.OpUnwrap:
		return g.genUnwrap(v)
	case ir.OpWrap:
		return g.genWrap(v)
	case ir.OpWrapNull:
		return g.genWrapNull(v)
	case ir.OpCopy:
		return g.genCopy(v)
	case ir.OpPhi:
		// Phi nodes are handled at control flow edges
		return nil
	case ir.OpCall:
		return g.genCall(v)
	case ir.OpReturn, ir.OpExit:
		// Handled by generateTerminator
		return nil
	default:
		return fmt.Errorf("unhandled IR operation: %s", v.Op)
	}
}

func (g *generator) genConst(v *ir.Value) error {
	if v.Type == nil {
		return fmt.Errorf("constant has no type")
	}

	// Constants without stack slots are materialized inline when used (in loadValue)
	// Only store to stack if this constant has a stack slot allocated
	offset, hasSlot := g.layout.Offsets[v]

	switch v.Type.(type) {
	case *ir.IntType:
		if !hasSlot {
			return nil // Will be materialized inline in loadValue
		}
		// Load immediate into register, then store to stack
		g.loadImmediate(v.AuxInt, "x9")
		g.storeToStack("x9", offset)

	case *ir.BoolType:
		if !hasSlot {
			return nil // Will be materialized inline in loadValue
		}
		val := 0
		if v.AuxInt != 0 {
			val = 1
		}
		g.emit("    mov x9, #%d", val)
		g.storeToStack("x9", offset)

	case *ir.StringType:
		// String constant - register in data section
		strIdx := -1
		for i, s := range g.strings {
			if s == v.AuxString {
				strIdx = i
				break
			}
		}
		if strIdx == -1 {
			// Add new string
			strIdx = len(g.strings)
			g.strings = append(g.strings, v.AuxString)
		}
		// Store string index in value for loadValue to use later
		v.AuxInt = int64(strIdx)
		if !hasSlot {
			return nil // Will be materialized inline in loadValue
		}
		g.emit("    adrp x9, _sl_str%d@PAGE", strIdx)
		g.emit("    add x9, x9, _sl_str%d@PAGEOFF", strIdx)
		g.storeToStack("x9", offset)

	default:
		return fmt.Errorf("unsupported constant type: %s", v.Type)
	}

	return nil
}

func (g *generator) genBinaryOp(v *ir.Value, op string) error {
	// Load operands
	g.loadValue(v.Args[0], "x10")
	g.loadValue(v.Args[1], "x11")

	// Perform operation
	g.emit("    %s x9, x10, x11", op)

	// Store result
	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genAdd(v *ir.Value) error {
	g.loadValue(v.Args[0], "x10")
	g.loadValue(v.Args[1], "x11")

	label := g.labels.NextLabel()

	// Check if signed or unsigned
	isSigned := true
	if intType, ok := v.Type.(*ir.IntType); ok {
		isSigned = intType.Signed
	}

	if isSigned {
		// Signed addition - check overflow flag
		g.emit("    adds x9, x10, x11")
		g.emit("    b.vc _sl_add_ok_%d", label)
		g.emitPanic(PanicOverflowAdd)
	} else {
		// Unsigned addition - check carry flag
		g.emit("    adds x9, x10, x11")
		g.emit("    b.cc _sl_add_ok_%d", label)
		g.emitPanic(PanicUnsignedOverAdd)
	}

	g.emit("_sl_add_ok_%d:", label)

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genSub(v *ir.Value) error {
	g.loadValue(v.Args[0], "x10")
	g.loadValue(v.Args[1], "x11")

	label := g.labels.NextLabel()

	// Check if signed or unsigned
	isSigned := true
	if intType, ok := v.Type.(*ir.IntType); ok {
		isSigned = intType.Signed
	}

	if isSigned {
		// Signed subtraction - check overflow flag
		g.emit("    subs x9, x10, x11")
		g.emit("    b.vc _sl_sub_ok_%d", label)
		g.emitPanic(PanicOverflowSub)
	} else {
		// Unsigned subtraction - check carry flag (carry clear = borrow)
		g.emit("    subs x9, x10, x11")
		g.emit("    b.cs _sl_sub_ok_%d", label) // cs = no borrow, cc = borrow
		g.emitPanic(PanicUnsignedUnderSub)
	}

	g.emit("_sl_sub_ok_%d:", label)

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genMul(v *ir.Value) error {
	g.loadValue(v.Args[0], "x10")
	g.loadValue(v.Args[1], "x11")

	label := g.labels.NextLabel()

	// Check if signed or unsigned
	isSigned := true
	if intType, ok := v.Type.(*ir.IntType); ok {
		isSigned = intType.Signed
	}

	// Perform multiplication
	g.emit("    mul x9, x10, x11")

	if isSigned {
		// For signed multiplication, get high 64 bits
		g.emit("    smulh x12, x10, x11")
		// Check: high should be sign extension of low (all 0s or all 1s)
		g.emit("    asr x13, x9, #63") // Arithmetic shift right by 63 = sign extension
		g.emit("    cmp x12, x13")
		g.emit("    b.eq _sl_mul_ok_%d", label)
		g.emitPanic(PanicOverflowMul)
	} else {
		// For unsigned multiplication, high 64 bits should be 0
		g.emit("    umulh x12, x10, x11")
		g.emit("    cbz x12, _sl_mul_ok_%d", label)
		g.emitPanic(PanicUnsignedOverMul)
	}

	g.emit("_sl_mul_ok_%d:", label)

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genDiv(v *ir.Value) error {
	g.loadValue(v.Args[0], "x10")
	g.loadValue(v.Args[1], "x11")

	// Check for division by zero
	label := g.labels.NextLabel()
	g.emit("    cbnz x11, _sl_div_ok_%d", label)
	g.emitPanic(PanicDivZero)
	g.emit("_sl_div_ok_%d:", label)

	// Signed division
	if intType, ok := v.Type.(*ir.IntType); ok && intType.Signed {
		g.emit("    sdiv x9, x10, x11")
	} else {
		g.emit("    udiv x9, x10, x11")
	}

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genMod(v *ir.Value) error {
	g.loadValue(v.Args[0], "x10")
	g.loadValue(v.Args[1], "x11")

	// Check for modulo by zero
	label := g.labels.NextLabel()
	g.emit("    cbnz x11, _sl_mod_ok_%d", label)
	g.emitPanic(PanicModZero)
	g.emit("_sl_mod_ok_%d:", label)

	// Modulo: a % b = a - (a / b) * b
	if intType, ok := v.Type.(*ir.IntType); ok && intType.Signed {
		g.emit("    sdiv x12, x10, x11")
	} else {
		g.emit("    udiv x12, x10, x11")
	}
	g.emit("    msub x9, x12, x11, x10")

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genNeg(v *ir.Value) error {
	g.loadValue(v.Args[0], "x10")
	g.emit("    neg x9, x10")

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genCmp(v *ir.Value, cond string) error {
	g.loadValue(v.Args[0], "x10")
	g.loadValue(v.Args[1], "x11")

	g.emit("    cmp x10, x11")
	g.emit("    cset x9, %s", cond)

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genStrEq(v *ir.Value) error {
	g.loadValue(v.Args[0], "x0")
	g.loadValue(v.Args[1], "x1")
	g.emit("    bl _sl_str_eq")

	offset := g.stackOffset(v)
	g.storeToStack("x0", offset)

	return nil
}

func (g *generator) genNot(v *ir.Value) error {
	g.loadValue(v.Args[0], "x10")
	g.emit("    cmp x10, #0")
	g.emit("    cset x9, eq")

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genAlloc(v *ir.Value) error {
	size := v.AuxInt

	// Call custom heap allocator
	g.emit("    mov x0, #%d", size)
	g.emit("    bl _sl_heap_alloc")

	// Store result pointer
	offset := g.stackOffset(v)
	g.storeToStack("x0", offset)

	return nil
}

func (g *generator) genLoad(v *ir.Value) error {
	// Load pointer
	g.loadValue(v.Args[0], "x10")

	// Check if loading a nullable value type (16 bytes: tag + value)
	if nullType, ok := v.Type.(*ir.NullableType); ok && !nullType.IsReferenceNullable() {
		offset := g.stackOffset(v)
		g.emit("    ldr x9, [x10]")     // tag
		g.storeToStack("x9", offset)
		g.emit("    ldr x9, [x10, #8]") // value
		g.storeToStack("x9", offset+8)
		return nil
	}

	// Use appropriate load instruction based on type size
	if v.Type != nil {
		switch v.Type.Size() {
		case 1:
			g.emit("    ldrb w9, [x10]")
		case 2:
			g.emit("    ldrh w9, [x10]")
		case 4:
			g.emit("    ldr w9, [x10]")
		default:
			g.emit("    ldr x9, [x10]")
		}
	} else {
		g.emit("    ldr x9, [x10]")
	}

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genLoadGlobal(v *ir.Value) error {
	label := "_sl_global_" + v.AuxString
	offset := g.stackOffset(v)
	g.emit("    adrp x9, %s@PAGE", label)
	g.emit("    add x9, x9, %s@PAGEOFF", label)
	g.emit("    ldr x9, [x9]")
	g.storeToStack("x9", offset)
	return nil
}

func (g *generator) genStoreGlobal(v *ir.Value) error {
	label := "_sl_global_" + v.AuxString
	g.loadValue(v.Args[0], "x9") // value to store
	g.emit("    adrp x10, %s@PAGE", label)
	g.emit("    add x10, x10, %s@PAGEOFF", label)
	g.emit("    str x9, [x10]")
	return nil
}

func (g *generator) genStore(v *ir.Value) error {
	// Load pointer
	g.loadValue(v.Args[0], "x10") // pointer

	// Check if storing a nullable value type (needs 16 bytes: tag + value)
	valueType := v.Args[1].Type
	if nullType, ok := valueType.(*ir.NullableType); ok && !nullType.IsReferenceNullable() {
		if argOffset, ok := g.layout.Offsets[v.Args[1]]; ok {
			g.loadFromStack("x9", argOffset)   // tag
			g.emit("    str x9, [x10]")
			g.loadFromStack("x9", argOffset+8) // value
			g.emit("    str x9, [x10, #8]")
		} else {
			g.loadValue(v.Args[1], "x9")
			g.emit("    str x9, [x10]")
			g.emit("    str xzr, [x10, #8]")
		}
		return nil
	}

	g.loadValue(v.Args[1], "x9") // value

	// Use appropriate store instruction based on type size
	if valueType != nil {
		switch valueType.Size() {
		case 1:
			g.emit("    strb w9, [x10]")
		case 2:
			g.emit("    strh w9, [x10]")
		case 4:
			g.emit("    str w9, [x10]")
		default:
			g.emit("    str x9, [x10]")
		}
	} else {
		g.emit("    str x9, [x10]")
	}

	return nil
}

func (g *generator) genFree(v *ir.Value) error {
	// Free memory: Args[0] = pointer, AuxInt = size
	g.loadValue(v.Args[0], "x0") // pointer to free
	g.loadImmediate(v.AuxInt, "x1") // size
	g.emit("    bl _sl_heap_free")
	return nil
}

func (g *generator) genMemCopy(v *ir.Value) error {
	// Load dest and src pointers
	g.loadValue(v.Args[0], "x10") // dest pointer
	g.loadValue(v.Args[1], "x11") // src pointer

	size := v.AuxInt

	// Copy in 8-byte chunks
	for offset := int64(0); offset < size; offset += 8 {
		g.emit("    ldr x9, [x11, #%d]", offset)
		g.emit("    str x9, [x10, #%d]", offset)
	}

	return nil
}

func (g *generator) genCopy(v *ir.Value) error {
	// Deep copy: allocate new memory, copy contents recursively
	// Args[0] is the source pointer

	// Get the element type
	var elemType ir.Type
	var size int64
	if v.Type != nil {
		if ptrType, ok := v.Type.(*ir.PtrType); ok && ptrType.Elem != nil {
			elemType = ptrType.Elem
			size = int64(ptrType.Elem.Size())
		}
	}
	if size == 0 {
		size = 8 // fallback
	}

	// Load source pointer and save to stack (will be clobbered by heap_alloc)
	g.loadValue(v.Args[0], "x11")
	g.emit("    str x11, [sp, #-16]!") // Push source pointer

	// Allocate new memory
	g.emit("    mov x0, #%d", size)
	g.emit("    bl _sl_heap_alloc")
	g.emit("    mov x10, x0") // x10 = new pointer

	// Restore source pointer
	g.emit("    ldr x11, [sp], #16") // Pop source pointer

	// Copy data in 8-byte chunks (shallow copy first)
	for offset := int64(0); offset < size; offset += 8 {
		g.emit("    ldr x9, [x11, #%d]", offset)
		g.emit("    str x9, [x10, #%d]", offset)
	}

	// Now handle deep copy for pointer fields in struct types
	if structType, ok := elemType.(*ir.StructType); ok {
		g.emitDeepCopyFields(structType.Fields, "x10", "x11")
	}

	// Store new pointer
	offset := g.stackOffset(v)
	g.storeToStack("x10", offset)

	return nil
}

// emitDeepCopyFields recursively copies pointer fields in a struct
// destReg and srcReg contain pointers to the destination and source structs
func (g *generator) emitDeepCopyFields(fields []ir.StructField, destReg, srcReg string) {
	for _, field := range fields {
		if ptrType, ok := field.Type.(*ir.PtrType); ok {
			// This field is a pointer - need to deep copy the pointed-to data
			fieldOffset := field.Offset
			pointedSize := ptrType.Elem.Size()

			// Save dest and src pointers
			g.emit("    stp %s, %s, [sp, #-16]!", destReg, srcReg)

			// Load source pointer field value
			g.emit("    ldr x9, [%s, #%d]", srcReg, fieldOffset)

			// Check if null - if so, skip copy (already copied null in shallow copy)
			label := g.labels.NextLabel()
			g.emit("    cbz x9, _sl_copy_field_done_%d", label)

			// Save field pointer value (will be clobbered by heap_alloc)
			g.emit("    str x9, [sp, #-16]!")

			// Allocate memory for the pointed-to type
			g.emit("    mov x0, #%d", pointedSize)
			g.emit("    bl _sl_heap_alloc")
			g.emit("    mov x12, x0") // x12 = new pointed-to memory

			// Restore source field pointer
			g.emit("    ldr x9, [sp], #16")

			// Copy the pointed-to data
			for offset := 0; offset < pointedSize; offset += 8 {
				g.emit("    ldr x13, [x9, #%d]", offset)
				g.emit("    str x13, [x12, #%d]", offset)
			}

			// Restore dest and src struct pointers
			g.emit("    ldp %s, %s, [sp], #16", destReg, srcReg)

			// Store new pointer in destination field
			g.emit("    str x12, [%s, #%d]", destReg, fieldOffset)

			// Handle recursive deep copy for nested struct pointers
			if nestedStruct, ok := ptrType.Elem.(*ir.StructType); ok && len(nestedStruct.Fields) > 0 {
				// Save current pointers
				g.emit("    stp %s, %s, [sp, #-16]!", destReg, srcReg)
				// Set up for recursive call - x12 is dest, x9 is src
				g.emitDeepCopyFields(nestedStruct.Fields, "x12", "x9")
				// Restore
				g.emit("    ldp %s, %s, [sp], #16", destReg, srcReg)
			}

			g.emit("_sl_copy_field_done_%d:", label)
		}
	}
}

func (g *generator) genFieldPtr(v *ir.Value) error {
	// Load struct pointer
	g.loadValue(v.Args[0], "x10")

	// Add field offset
	fieldOffset := v.AuxInt
	if fieldOffset != 0 {
		g.emit("    add x9, x10, #%d", fieldOffset)
	} else {
		g.emit("    mov x9, x10")
	}

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genIndexPtr(v *ir.Value) error {
	// Load array pointer and index
	g.loadValue(v.Args[0], "x10")
	g.loadValue(v.Args[1], "x11")

	// Get array length from type if available, for bounds checking
	var arrayLen int64 = -1
	if ptrType, ok := v.Args[0].Type.(*ir.PtrType); ok {
		if arrType, ok := ptrType.Elem.(*ir.ArrayType); ok {
			arrayLen = int64(arrType.Len)
		}
	}

	// Bounds checking (if array length is known)
	if arrayLen >= 0 {
		label := g.labels.NextLabel()

		// Check index < 0
		g.emit("    cmp x11, #0")
		g.emit("    blt _sl_bounds_fail_%d", label)

		// Check index >= len
		g.loadImmediate(arrayLen, "x12")
		g.emit("    cmp x11, x12")
		g.emit("    blt _sl_bounds_ok_%d", label)

		// Bounds check failed
		g.emit("_sl_bounds_fail_%d:", label)
		g.emitPanic(PanicBounds)

		g.emit("_sl_bounds_ok_%d:", label)
	}

	// Calculate element size from type
	elemSize := int64(8) // default
	if ptrType, ok := v.Type.(*ir.PtrType); ok {
		elemSize = int64(ptrType.Elem.Size())
	}

	// Calculate offset: base + index * elemSize
	if elemSize == 8 {
		g.emit("    add x9, x10, x11, lsl #3")
	} else if elemSize == 4 {
		g.emit("    add x9, x10, x11, lsl #2")
	} else if elemSize == 2 {
		g.emit("    add x9, x10, x11, lsl #1")
	} else if elemSize == 1 {
		g.emit("    add x9, x10, x11")
	} else {
		g.emit("    mov x12, #%d", elemSize)
		g.emit("    mul x12, x11, x12")
		g.emit("    add x9, x10, x12")
	}

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genArrayLen(v *ir.Value) error {
	// For now, array length is stored as metadata
	// This depends on how arrays are represented
	// For a simple fixed-size array, length is known at compile time
	if arrType, ok := v.Args[0].Type.(*ir.PtrType); ok {
		if arr, ok := arrType.Elem.(*ir.ArrayType); ok {
			g.emit("    mov x9, #%d", arr.Len)
			offset := g.stackOffset(v)
			g.storeToStack("x9", offset)
			return nil
		}
	}

	// Fallback: load length from runtime header (first 8 bytes)
	g.loadValue(v.Args[0], "x10")
	g.emit("    ldr x9, [x10, #-8]") // Length stored before array data
	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genIsNull(v *ir.Value) error {
	g.loadValue(v.Args[0], "x10")
	g.emit("    cmp x10, #0")
	g.emit("    cset x9, eq")

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genUnwrap(v *ir.Value) error {
	// For reference nullables, just load the pointer
	// For value nullables, load the value part (skip the tag)
	if nullType, ok := v.Args[0].Type.(*ir.NullableType); ok {
		if nullType.IsReferenceNullable() {
			// Pointer is the value - just load it
			g.loadValue(v.Args[0], "x9")
		} else {
			// Value type nullable - need to load from offset+8
			// Get the address of the nullable storage, then load value from offset+8
			if argOffset, ok := g.layout.Offsets[v.Args[0]]; ok {
				g.loadFromStack("x9", argOffset+8)
			} else {
				// Shouldn't happen - nullable values should always be on stack
				g.loadValue(v.Args[0], "x9")
			}
		}
	} else {
		g.loadValue(v.Args[0], "x9")
	}

	offset := g.stackOffset(v)
	g.storeToStack("x9", offset)

	return nil
}

func (g *generator) genWrap(v *ir.Value) error {
	// Wrap a value as nullable (set tag to 1 = not null)
	g.loadValue(v.Args[0], "x10")

	if nullType, ok := v.Type.(*ir.NullableType); ok {
		if nullType.IsReferenceNullable() {
			// Pointer value - just use the pointer
			g.emit("    mov x9, x10")
			offset := g.stackOffset(v)
			g.storeToStack("x9", offset)
		} else {
			// Value type - store tag (1) and value
			offset := g.stackOffset(v)
			g.emit("    mov x9, #1")
			g.storeToStack("x9", offset)
			g.storeToStack("x10", offset+8)
		}
	} else {
		g.emit("    mov x9, x10")
		offset := g.stackOffset(v)
		g.storeToStack("x9", offset)
	}

	return nil
}

func (g *generator) genWrapNull(v *ir.Value) error {
	// Create a null value (tag = 0 or pointer = 0)
	offset := g.stackOffset(v)
	g.emit("    mov x9, xzr")
	g.storeToStack("x9", offset)

	if nullType, ok := v.Type.(*ir.NullableType); ok {
		if !nullType.IsReferenceNullable() {
			// Also zero the value part
			g.storeToStack("x9", offset+8)
		}
	}

	return nil
}

func (g *generator) genCall(v *ir.Value) error {
	funcName := v.AuxString

	// Handle built-in functions specially
	switch funcName {
	case "print":
		return g.genPrint(v)
	case "exit":
		return g.genExit(v)
	case "len":
		// Should have been lowered to OpArrayLen
		return g.genArrayLen(v)
	case "sleep":
		return g.genSleep(v)
	case "assert":
		return g.genAssert(v)
	}

	// Regular function call
	// Load arguments into registers. Nullable value-type args use two registers (tag + value).
	regIdx := 0
	for _, arg := range v.Args {
		if nullType, ok := arg.Type.(*ir.NullableType); ok && !nullType.IsReferenceNullable() {
			// Nullable value type: load tag into regIdx, value into regIdx+1
			if argOffset, ok := g.layout.Offsets[arg]; ok {
				g.loadFromStack(fmt.Sprintf("x%d", regIdx), argOffset)   // tag
				g.loadFromStack(fmt.Sprintf("x%d", regIdx+1), argOffset+8) // value
			} else {
				g.emit("    mov x%d, #0", regIdx)
				g.emit("    mov x%d, #0", regIdx+1)
			}
			regIdx += 2
		} else {
			if regIdx < 8 {
				g.loadValue(arg, fmt.Sprintf("x%d", regIdx))
			}
			regIdx++
		}
	}

	// Call function
	g.emit("    bl _%s", funcName)

	// Store return value if any
	if v.Type != nil && !v.Type.Equal(ir.TypeVoid) {
		offset := g.stackOffset(v)
		// Check if returning a value-type nullable (uses x0 + x1)
		if nullType, ok := v.Type.(*ir.NullableType); ok && !nullType.IsReferenceNullable() {
			g.storeToStack("x0", offset)   // tag
			g.storeToStack("x1", offset+8) // value
		} else {
			g.storeToStack("x0", offset)
		}
	}

	return nil
}

func (g *generator) genPrint(v *ir.Value) error {
	if len(v.Args) == 0 {
		return nil
	}

	arg := v.Args[0]
	g.loadValue(arg, "x0")

	// Call appropriate print helper based on type
	switch arg.Type.(type) {
	case *ir.IntType:
		g.emit("    bl _sl_print_int")
	case *ir.StringType:
		g.emit("    bl _sl_print_str")
	case *ir.BoolType:
		g.emit("    bl _sl_print_bool")
	default:
		// Default to integer
		g.emit("    bl _sl_print_int")
	}

	return nil
}

func (g *generator) genExit(v *ir.Value) error {
	if len(v.Args) > 0 {
		g.loadValue(v.Args[0], "x0")
	} else {
		g.emit("    mov x0, #0")
	}
	// Direct exit syscall
	g.emit("    mov x16, #1")
	g.emit("    svc #0")

	return nil
}

func (g *generator) genSleep(v *ir.Value) error {
	// sleep takes nanoseconds, use select syscall with timeval
	if len(v.Args) > 0 {
		g.loadValue(v.Args[0], "x10")

		// Allocate timeval struct on stack (16 bytes: tv_sec + tv_usec)
		g.emit("    sub sp, sp, #16")

		// Load 1,000,000,000 for division to get seconds
		g.emit("    mov x11, #0xCA00")
		g.emit("    movk x11, #0x3B9A, lsl #16")

		// tv_sec = ns / 1,000,000,000
		g.emit("    sdiv x12, x10, x11")
		g.emit("    str x12, [sp]")

		// remainder = ns % 1,000,000,000
		g.emit("    msub x13, x12, x11, x10")

		// tv_usec = remainder / 1000 (convert ns to microseconds)
		g.emit("    mov x11, #1000")
		g.emit("    sdiv x13, x13, x11")
		g.emit("    str x13, [sp, #8]")

		// Call select(0, NULL, NULL, NULL, &timeval)
		g.emit("    mov x0, #0")
		g.emit("    mov x1, #0")
		g.emit("    mov x2, #0")
		g.emit("    mov x3, #0")
		g.emit("    mov x4, sp")
		g.emit("    mov x16, #93") // SYS_select
		g.emit("    svc #0x80")

		// Clean up stack
		g.emit("    add sp, sp, #16")
	}

	return nil
}

func (g *generator) genAssert(v *ir.Value) error {
	if len(v.Args) < 2 {
		return nil
	}

	// Generate unique labels for this assertion
	labelNum := g.labels.NextLabel()
	passLabel := fmt.Sprintf("_assert_pass_%d", labelNum)
	strlenLabel := fmt.Sprintf("_assert_strlen_%d", labelNum)
	strlenDoneLabel := fmt.Sprintf("_assert_strlen_done_%d", labelNum)

	// Load condition (first arg)
	g.loadValue(v.Args[0], "x10")

	// If condition is true (non-zero), skip to pass label
	g.emit("    cbnz x10, %s", passLabel)

	// Condition is false - print message to stderr and exit with code 1
	// Load message string pointer (second arg)
	g.loadValue(v.Args[1], "x10")

	// Print "assertion failed: " prefix to stderr
	g.emit("    adrp x1, _sl_assert_prefix@PAGE")
	g.emit("    add x1, x1, _sl_assert_prefix@PAGEOFF")
	g.emit("    mov x0, #2")  // stderr
	g.emit("    mov x2, #18") // length of "assertion failed: "
	g.emit("    mov x16, #4") // write syscall
	g.emit("    svc #0")

	// Calculate message length and print it
	g.emit("    mov x19, x10") // Save string pointer
	g.emit("    mov x11, x10")
	g.emit("%s:", strlenLabel)
	g.emit("    ldrb w12, [x11]")
	g.emit("    cbz w12, %s", strlenDoneLabel)
	g.emit("    add x11, x11, #1")
	g.emit("    b %s", strlenLabel)
	g.emit("%s:", strlenDoneLabel)
	g.emit("    sub x2, x11, x19") // length = end - start

	// Write message to stderr
	g.emit("    mov x0, #2")   // stderr
	g.emit("    mov x1, x19")  // message pointer
	g.emit("    mov x16, #4")  // write syscall
	g.emit("    svc #0")

	// Print newline
	g.emit("    adrp x1, _sl_newline@PAGE")
	g.emit("    add x1, x1, _sl_newline@PAGEOFF")
	g.emit("    mov x0, #2")  // stderr
	g.emit("    mov x2, #1")  // length of newline
	g.emit("    mov x16, #4") // write syscall
	g.emit("    svc #0")

	// Exit with code 1
	g.emit("    mov x0, #1")
	g.emit("    mov x16, #1") // exit syscall
	g.emit("    svc #0")

	// Pass label - assertion succeeded
	g.emit("%s:", passLabel)

	return nil
}

func (g *generator) generateTerminator(block *ir.Block) error {
	switch block.Kind {
	case ir.BlockPlain:
		if len(block.Succs) > 0 {
			succ := block.Succs[0]
			g.emitPhiCopies(block, succ)
			g.emit("    b %s", g.labels.BlockLabel(succ))
		}

	case ir.BlockIf:
		if block.Control == nil || len(block.Succs) < 2 {
			return fmt.Errorf("malformed if block")
		}

		g.loadValue(block.Control, "x9")
		thenBlock := block.Succs[0]
		elseBlock := block.Succs[1]

		thenLabel := g.labels.BlockLabel(thenBlock)
		elseLabel := g.labels.BlockLabel(elseBlock)

		// Check if then block needs phi copies
		thenNeedsPhiCopy := g.blockNeedsPhiCopy(block, thenBlock)

		if thenNeedsPhiCopy {
			// Use intermediate label for phi copies before jumping to then block
			phiCopyLabel := fmt.Sprintf("%s_phi_%d", thenLabel, g.labels.NextLabel())
			g.emit("    cbnz x9, %s", phiCopyLabel)
			g.emitPhiCopies(block, elseBlock)
			g.emit("    b %s", elseLabel)
			// Emit phi copies for then path
			g.emit("%s:", phiCopyLabel)
			g.emitPhiCopies(block, thenBlock)
			g.emit("    b %s", thenLabel)
		} else {
			// No phi copies needed for then, branch directly
			g.emit("    cbnz x9, %s", thenLabel)
			g.emitPhiCopies(block, elseBlock)
			g.emit("    b %s", elseLabel)
		}

	case ir.BlockReturn:
		g.emitReturnValue(block)
		g.emitEpilogue()

	case ir.BlockExit:
		// Find exit value
		var exitVal *ir.Value
		for _, v := range block.Values {
			if v.Op == ir.OpExit {
				exitVal = v
				break
			}
		}

		if exitVal != nil && len(exitVal.Args) > 0 {
			g.loadValue(exitVal.Args[0], "x0")
		} else {
			g.emit("    mov x0, #0")
		}
		g.emit("    bl _exit")
	}

	return nil
}

func (g *generator) emitPhiCopies(from *ir.Block, to *ir.Block) {
	// Emit copies for phi nodes in the target block
	for _, v := range to.Values {
		if v.Op != ir.OpPhi {
			break
		}

		// Find the value from this predecessor
		for _, phiArg := range v.PhiArgs {
			if phiArg.From == from {
				if phiArg.Value != nil {
					offset := g.stackOffset(v)
					// Check if this is a value-type nullable that needs 16-byte copy
					if nullType, ok := v.Type.(*ir.NullableType); ok && !nullType.IsReferenceNullable() {
						// Copy both tag and value (16 bytes total)
						if srcOffset, ok := g.layout.Offsets[phiArg.Value]; ok {
							g.loadFromStack("x9", srcOffset)
							g.storeToStack("x9", offset)
							g.loadFromStack("x9", srcOffset+8)
							g.storeToStack("x9", offset+8)
						} else {
							// Source not on stack - shouldn't happen for nullable values
							g.loadValue(phiArg.Value, "x9")
							g.storeToStack("x9", offset)
						}
					} else {
						// Regular 8-byte value
						g.loadValue(phiArg.Value, "x9")
						g.storeToStack("x9", offset)
					}
				}
				break
			}
		}
	}
}

// blockNeedsPhiCopy checks if there are any phi copies needed when going from 'from' to 'to'.
func (g *generator) blockNeedsPhiCopy(from *ir.Block, to *ir.Block) bool {
	for _, v := range to.Values {
		if v.Op != ir.OpPhi {
			break
		}
		for _, phiArg := range v.PhiArgs {
			if phiArg.From == from && phiArg.Value != nil {
				return true
			}
		}
	}
	return false
}

func (g *generator) loadValue(v *ir.Value, reg string) {
	if v == nil {
		g.emit("    mov %s, #0", reg) // Fallback for nil values
		return
	}
	if offset, ok := g.layout.Offsets[v]; ok {
		g.loadFromStack(reg, offset)
	} else {
		// Value might be a constant that wasn't stored
		if v.Op == ir.OpConst {
			switch v.Type.(type) {
			case *ir.IntType, *ir.BoolType:
				g.loadImmediate(v.AuxInt, reg)
			case *ir.StringType:
				// String index stored in AuxInt
				strIdx := v.AuxInt
				g.emit("    adrp %s, _sl_str%d@PAGE", reg, strIdx)
				g.emit("    add %s, %s, _sl_str%d@PAGEOFF", reg, reg, strIdx)
			default:
				g.emit("    mov %s, #0", reg) // Fallback
			}
		}
	}
}

// loadImmediate loads an integer immediate into a register.
// Handles large values that don't fit in a single mov instruction.
func (g *generator) loadImmediate(val int64, reg string) {
	// Treat the value as unsigned bits for encoding
	uval := uint64(val)

	if uval < 65536 {
		// Small positive value - single mov
		g.emit("    mov %s, #%d", reg, uval)
	} else {
		// Large constant needs movz + movk instructions
		// First chunk with movz
		g.emit("    movz %s, #%d", reg, uval&0xFFFF)

		// Remaining chunks with movk
		if (uval>>16)&0xFFFF != 0 {
			g.emit("    movk %s, #%d, lsl #16", reg, (uval>>16)&0xFFFF)
		}
		if (uval>>32)&0xFFFF != 0 {
			g.emit("    movk %s, #%d, lsl #32", reg, (uval>>32)&0xFFFF)
		}
		if (uval>>48)&0xFFFF != 0 {
			g.emit("    movk %s, #%d, lsl #48", reg, (uval>>48)&0xFFFF)
		}
	}
}

func (g *generator) emit(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	g.builder.WriteString(line)
	g.builder.WriteString("\n")
}

// emitPanicCall emits a call to the panic helper with error message and function name.
// panicLabel is the data section label for the error message (e.g., "_sl_panic_div_zero")
// msgLen is the length of the error message including newline
// emitPanic emits a call to the panic handler with the given panic message.
// The message length is auto-computed from the panicMessage.
func (g *generator) emitPanic(p panicMessage) {
	g.emit("    adrp x0, %s@PAGE", p.Label)
	g.emit("    add x0, x0, %s@PAGEOFF", p.Label)
	g.emit("    mov x1, #%d", p.Len())
	g.emit("    adrp x2, _sl_fn_name_%s@PAGE", g.fn.Name)
	g.emit("    add x2, x2, _sl_fn_name_%s@PAGEOFF", g.fn.Name)
	g.emit("    mov x3, #%d", len(g.fn.Name))
	g.emit("    bl _sl_panic")
}

// storeToStack stores a register to a stack offset, handling large offsets.
// Uses x8 as a scratch register for address calculation if needed.
func (g *generator) storeToStack(reg string, offset int) {
	if offset >= -255 && offset <= 255 {
		// Small offset - use regular str (slasm supports signed offsets in this range)
		g.emit("    str %s, [x29, #%d]", reg, offset)
	} else if offset < 0 {
		// Large negative offset - use sub with positive value
		g.loadImmediate(int64(-offset), "x8")
		g.emit("    sub x8, x29, x8")
		g.emit("    str %s, [x8]", reg)
	} else {
		// Large positive offset - use add
		g.loadImmediate(int64(offset), "x8")
		g.emit("    add x8, x29, x8")
		g.emit("    str %s, [x8]", reg)
	}
}

// loadFromStack loads a register from a stack offset, handling large offsets.
// Uses x8 as a scratch register for address calculation if needed.
func (g *generator) loadFromStack(reg string, offset int) {
	if offset >= -255 && offset <= 255 {
		// Small offset - use regular ldr
		g.emit("    ldr %s, [x29, #%d]", reg, offset)
	} else if offset < 0 {
		// Large negative offset - use sub with positive value
		g.loadImmediate(int64(-offset), "x8")
		g.emit("    sub x8, x29, x8")
		g.emit("    ldr %s, [x8]", reg)
	} else {
		// Large positive offset - use add
		g.loadImmediate(int64(offset), "x8")
		g.emit("    add x8, x29, x8")
		g.emit("    ldr %s, [x8]", reg)
	}
}

// emitRaw writes a line without format processing.
func (g *generator) emitRaw(line string) {
	g.builder.WriteString(line)
	g.builder.WriteString("\n")
}

// escapeString escapes special characters for assembly string literals.
func escapeString(s string) string {
	var result strings.Builder
	for _, c := range s {
		switch c {
		case '\n':
			result.WriteString("\\n")
		case '\t':
			result.WriteString("\\t")
		case '\r':
			result.WriteString("\\r")
		case '"':
			result.WriteString("\\\"")
		case '\\':
			result.WriteString("\\\\")
		default:
			result.WriteRune(c)
		}
	}
	return result.String()
}
