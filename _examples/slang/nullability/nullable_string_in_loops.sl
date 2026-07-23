// @test: exit_code=0
// @test: stdout=m1\nm1\nm1\n3\n
// Regression: reading a string? binding inside a loop yields an incomplete
// phi whose type is not yet known; consulting that nil SSA type re-wrapped
// the already-boxed value and ran the string copy on a box pointer — an
// infinite loop at runtime. Identifier reads now take their type from the
// semantic type, so loop bodies can store and pass string? bindings freely.
H = struct { var s: string?  val n: s64 }

use = (s: string?) -> s64 {
    print(s ?: "-")
    if s == null {
        return 0
    }
    return 1
}

main = () {
    val m: string? = "m${1}"
    var total = 0
    for (var i = 0; i < 3; i = i + 1) {
        val h = H{ m, i }        // literal store of the binding, per iteration
        total = total + use(m)   // and passed as an argument
    }
    print(total)                 // 3
}
