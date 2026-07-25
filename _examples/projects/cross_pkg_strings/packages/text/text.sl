// String literals in this package occupy the package's own constant pool,
// numbered from zero. The root package numbers its pool from zero as well, so
// combining the two IR programs must remap these indices onto a merged pool
// rather than appending the functions and dropping the pool.

tag = () -> string {
    return "PKG_ZERO"
}

other_tag = () -> string {
    return "PKG_ONE"
}

// A literal also present in the root package must dedupe to a single pool
// entry without disturbing either side's references.
shared = () -> string {
    return "SHARED"
}

// The string builtins, called from inside a mangled (non-root) package.
head = (s: string, n: s64) -> string {
    return substr(s, 0, n)
}

first_char = (s: string) -> string {
    return chr(s[0])
}

label = (s: string) -> string {
    return "pkg:" + s
}

// A top-level binding in a non-root package: stored as a mangled global and
// read from a function other than main.
val BRACKET = "<>"

bracket = () -> string {
    return BRACKET
}
