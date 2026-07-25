// @test: exit_code=0
// @test: stdout=8\n3\n10\n99\n2\n0\n1\n2\n2\nlocal\nglobal\n
// A top-level binding is stored in .data, but a same-named function-local
// binding is not: parameters, locals, block locals and for-init variables all
// shadow it. Resolving those to the global would make an unrelated local read
// and overwrite shared state that every other function can see.

val NAME = "global"
var COUNTER = 0

bump = () {
    COUNTER = COUNTER + 1
}

// A parameter named like a top-level binding
double = (COUNTER: s64) -> s64 {
    return COUNTER * 2
}

// A local named like a top-level binding
local_wins = () -> s64 {
    val COUNTER = 3
    return COUNTER
}

// A local inside a nested block shadows only until the block ends
block_scoped = () -> s64 {
    if true {
        val COUNTER = 99
        print(COUNTER)
    }
    return COUNTER
}

// A for-init binding is scoped to the loop
loop_scoped = () -> s64 {
    for (var COUNTER = 0; COUNTER < 3; COUNTER = COUNTER + 1) {
        print(COUNTER)
    }
    return COUNTER
}

shadow_string = () -> string {
    val NAME = "local"
    return NAME
}

main = () {
    bump()
    bump()

    print(double(4))
    print(local_wins())

    // The global is untouched by any of the shadowing above
    print(COUNTER + 8)

    print(block_scoped())
    print(loop_scoped())
    print(shadow_string())
    print(NAME)
}
