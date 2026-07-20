// @test: exit_code=0
// @test: stdout=11\n1\n2\n
// Regression: a vec binding used as a struct-literal field initializer must be
// deep-copied into the field (copy-on-store, like string fields). Aliasing the
// header made both the binding and the struct free the same buffer — a double
// free — and let a returned struct carry a dangling vec.
Holder = struct { var v: vec  val tag: s64 }

make = () -> Holder {
    var v = vec()
    push(v, 11)
    return Holder{ v, 1 }
}

main = () {
    val h = make()
    print(get(h.v, 0))   // 11 — vec survived the return intact

    var ve = vec()
    push(ve, 7)
    val h2 = Holder{ ve, 2 }
    push(ve, 8)          // grows only the binding
    print(len(h2.v))     // 1 — field owns an independent copy
    print(len(ve))       // 2
}
