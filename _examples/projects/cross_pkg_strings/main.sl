// @test: exit_code=0
// @test: stdout=PKG_ZERO\nPKG_ONE\nMAIN_ZERO\nMAIN_ONE\nabc\nZ\npkg:x\nSHARED\nSHARED\ntrue\n<>\n
// Each package numbers its string-constant pool from zero, so the root and the
// imported package both reference _sl_str0, _sl_str1, ... for different
// literals. Combining the IR programs must merge the pools and remap the
// references; appending functions alone drops every literal past the first
// package's pool. This test only fails at link or print time, so it needs a
// package whose literals interleave with the root's.
text = import "text"

main = () {
    // The package's own literals, at its pool indices 0 and 1
    print(text.tag())
    print(text.other_tag())

    // The root's literals, which occupy the same indices in its pool
    print("MAIN_ZERO")
    print("MAIN_ONE")

    // The string builtins reached through a mangled package function
    print(text.head("abcdef", 3))
    print(text.first_char("Zebra"))
    print(text.label("x"))

    // A literal present in both pools must dedupe to one entry and still be
    // correct from either side
    print(text.shared())
    print("SHARED")
    print(text.shared() == "SHARED")

    // A top-level val declared in the package, read from a package function
    print(text.bracket())
}
