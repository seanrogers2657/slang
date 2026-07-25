// @test: exit_code=0
// @test: stdout=6\n2\n5\n2\n6\n
// A local that shadows a top-level heap-owning binding introduces two distinct
// allocations under one name. Scope-exit cleanup frees an owned binding by
// reading its value back, so the top-level entry must read its .data global
// directly — a by-name read would resolve through the shadow to the local's
// value, freeing that buffer twice and leaking the global's.
//
// The runtime's balanced-heap assertion at exit is what catches this: a double
// free drives the balance negative, a leak drives it positive.

val G = "gg"
var V = "vv"

main = () {
    // Shadow with a freshly allocated (non-constant) string
    val G = "local" + "!"
    print(len(G))

    // The global is still intact and freed exactly once
    print(len(read_g()))

    var V = "loc" + "al"
    print(len(V))
    print(len(read_v()))

    print(nested())
}

// A heap local shadowing the top-level binding inside a nested scope. (A local
// may shadow a top-level binding; shadowing an enclosing *local* is rejected by
// the analyzer, so the shadow has to live in its own function.)
nested = () -> s64 {
    var total = 0
    if true {
        val G = "xyz" + "w"
        total = total + len(G)
    }
    return total + len(G)
}

read_g = () -> string {
    return G
}

read_v = () -> string {
    return V
}
