// @test: exit_code=0
// @test: stdout=pre-\n42\ntrue\npre-x\ntrue\ntrue\n2\n
// A top-level binding lives outside any function, so it is stored in .data and
// read through a global rather than as an SSA variable. SSA definitions are
// function-local, so an SSA-backed top-level binding would be visible only to
// the function the top-level statements were injected into (main) — any other
// function reading it fails IR validation with "used before definition".
//
// This applies to `val` exactly as much as to `var`: immutability is a semantic
// rule, not a storage one.

val PREFIX = "pre-"
val ANSWER = 42
val ENABLED = true
var COUNTER = 0

read_prefix = () -> string {
    return PREFIX
}

read_answer = () -> s64 {
    return ANSWER
}

read_enabled = () -> bool {
    return ENABLED
}

label = (s: string) -> string {
    return PREFIX + s
}

// A second reader of the same binding
also_read_prefix = () -> string {
    return PREFIX
}

bump = () {
    COUNTER = COUNTER + 1
}

main = () {
    print(read_prefix())
    print(read_answer())
    print(read_enabled())
    print(label("x"))

    // The same immutable binding read from two different functions
    print(read_prefix() == PREFIX)
    print(also_read_prefix() == read_prefix())

    // Mutable top-level state still round-trips across functions
    bump()
    bump()
    print(COUNTER)
}
