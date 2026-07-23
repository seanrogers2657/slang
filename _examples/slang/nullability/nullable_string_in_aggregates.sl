// @test: exit_code=0
// @test: stdout=a=tw3\ns=tw3\n2=o9\nfalse\n
// Regression: storing a string? binding into an aggregate — struct literal
// field, class field, field assignment, array element — must copy the box and
// its contents (copy-on-store), not alias them. Aliasing made both the binding
// and the aggregate free the same box+buffer.
H1 = struct { var a: string?  val n: s64 }
Holder = struct { var s: string?  val tag: s64 }

main = () {
    val s: string? = "tw${3}"
    val h = H1{ s, 1 }            // literal field copies
    print("a=${h.a}")             // a=tw3
    print("s=${s}")               // s=tw3 — binding still owns its own

    var other: string? = "o${9}"
    val h2 = Holder{ null, 1 }
    h2.s = other                  // field assign copies
    print("2=${h2.s}")            // 2=o9

    val m1: string? = "m${1}"
    var arr = [m1, m1]            // array elements copy
    print(arr[0] == null)         // false
}
