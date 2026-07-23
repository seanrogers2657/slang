// @test: exit_code=0
// @test: stdout=a1\ntrue\nfalse\n5\nfalse\n
// Regression: a non-null string? owns its heap buffer and must free it — at
// scope exit, when reassigned (to null or a new value), when held in a struct
// field, and when bound from another string? (copy-on-store: box and contents
// are copied, so both bindings free their own). Each of these used to leak the
// buffer, or double-free it once the leak was fixed naively.
Holder = struct { var s: string?  val tag: s64 }

main = () {
    val name = "a${1}"
    val s: string? = name     // copies: binding and s own separate buffers
    print(name)               // a1

    var t: string? = "b${2}"
    t = "c${3}"               // old buffer freed
    t = null                  // and again
    print(t == null)          // true

    var a: string? = "d${4}"
    var b = a                 // copy-on-store: independent box + buffer
    a = null                  // frees only a's copy
    print(b == null)          // false

    val h = Holder{ "e${5}", 5 }
    print(h.tag)              // 5 — field's buffer freed with the struct
    print(s == null)          // false
}
