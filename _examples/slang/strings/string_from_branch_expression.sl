// @test: exit_code=0
// @test: stdout=n=42\nn=42\nn=7\n
// Regression: a string yielded from an if/when branch is copied at the branch
// (value semantics) so the new binding owns an independent buffer. Aliasing
// the source string used to double-free at scope exit.
main = () {
    val n = 42
    val name = "n=${n}"
    val c = true

    val s = if c { name } else { "y" }
    print(s)      // n=42 — independent copy
    print(name)   // n=42 — original still owns its buffer

    val m = 7
    val heap = "n=${m}"
    val t = when {
        c -> heap
        else -> "z"
    }
    print(t)      // n=7
}
