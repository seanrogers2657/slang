// @test: exit_code=0
// @test: stdout=2\n1\n1\n2\n1\n
// Regression: an array literal element that is a struct value read from a
// binding or a nested field must be deep-copied so the array owns independent
// storage. A struct read from a nested field (h.b) is not a plain identifier,
// so the old alias-marking missed it and its inner vec was freed twice (once by
// the array element, once by the enclosing struct). A struct read from a plain
// variable was alias-marked (balanced) but aliased rather than copied.
Bag = struct { var s: string  var v: vec }
Holder = struct { var b: Bag  var id: s64 }

main = () {
    // Nested-field source: [h.b].
    var h = Holder{ Bag{ "orig", vec() }, 1 }
    push(h.b.v, 10)
    var arr = [h.b]
    push(arr[0].v, 20)
    print(len(arr[0].v))   // 2 — the copy grew
    print(len(h.b.v))      // 1 — original untouched (independent copy, no double free)

    // Plain-variable source: [r] must also be an independent copy now.
    var r = Bag{ "r", vec() }
    push(r.v, 1)
    var arr2 = [r]
    print(len(arr2[0].v))  // 1
    push(r.v, 2)
    print(len(r.v))        // 2
    print(len(arr2[0].v))  // 1 — element unaffected by the source's growth
}
