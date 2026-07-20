// @test: exit_code=0
// @test: stdout=m3\nm3\nm1\nm1\n
// Regression: a nested literal whose inner literal copies a boxed string?
// binding splits blocks mid-field-loop; the literal generators must refresh
// their builder after generating each field or the following stores land in a
// stale block (IR validation failure, or a miscompiled runaway copy).
Inner = struct { var s: string?  val n: s64 }
Outer = struct { var i: Inner  val z: s64 }

main = () {
    val m: string? = "m${3}"
    val o = Outer{ Inner{ m, 1 }, 2 }
    print(o.i.s ?: "-")     // m3
    print(m ?: "-")         // m3 — binding still owns its own

    val m2: string? = "m${1}"
    var arr = [Inner{ m2, 1 }, Inner{ m2, 2 }]
    print(arr[0].s ?: "-")  // m1
    print(arr[1].s ?: "-")  // m1
}
