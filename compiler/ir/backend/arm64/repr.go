package arm64

import "github.com/seanrogers2657/slang/compiler/ir"

// repr describes how a value of a given IR type is laid out and moved by the
// backend. It exists so codegen sites branch on a single classified descriptor
// instead of re-deriving type structure ("if it's a flat nullable...") at every
// load, store, phi, call, and return.
//
// The movement concern is representation-agnostic: a value occupies `words`
// 8-byte words, and anything wider than one word (`multiWord`) is copied
// word-by-word, returned in a register tuple (x0..xN), and passed by pointer.
// Only the nullable wrap/unwrap/null-check ops care about the tag+payload layout
// itself, which `flatNullable` flags.
type repr struct {
	// words is the number of 8-byte words the value occupies (always >= 1).
	words int
	// flatNullable is true for the flat tag+payload nullable layout
	// (tag at byte 0, payload at byte 8). See ir.NullableType.IsFlat.
	flatNullable bool
}

// reprOf classifies an IR type's representation.
//
// Crucially, the *value representation* size is not Type.Size(): a struct,
// class, array, or string is held as an 8-byte pointer in a slot/register, even
// though its Size() reports the (larger) pointee size. So the only values that
// occupy more than one word in a slot are flat nullables, whose tag+payload
// aggregate is stored inline. Everything else is a single word.
func reprOf(t ir.Type) repr {
	if nt, ok := t.(*ir.NullableType); ok && nt.IsFlat() {
		words := nt.Size() / 8
		if words < 1 {
			words = 1
		}
		return repr{words: words, flatNullable: true}
	}
	// 128-bit integers occupy two words and are moved word-by-word, like flat
	// nullables. This is what lets the existing multi-word param/return/call/
	// load/store/phi plumbing carry s128/u128 values.
	if it, ok := t.(*ir.IntType); ok && it.Bits >= 128 {
		return repr{words: it.Bits / 64}
	}
	return repr{words: 1}
}

// multiWord reports whether the value is wider than a single register and must
// be moved word-by-word (rather than through one load/store).
func (r repr) multiWord() bool { return r.words > 1 }

// bytes returns the value's size in bytes (words rounded form).
func (r repr) bytes() int { return r.words * 8 }
