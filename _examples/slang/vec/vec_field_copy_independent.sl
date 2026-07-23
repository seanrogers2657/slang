// @test: exit_code=0
// @test: stdout=3\n2\n10\n99\n
// Regression: .copy() of a `new`-allocated aggregate with a vec field must
// deep-copy the vec so the copy owns an independent buffer. A shallow pointer
// copy makes both instances free the same buffer (a double free) and lets
// mutations leak across. After the copy, growing/mutating one must not affect
// the other, and the heap must stay balanced.
Bag = struct { var items: vec  val tag: s64 }

main = () {
    val a = new Bag{ vec(), 1 }
    push(a.items, 10)
    push(a.items, 20)

    val b = a.copy()         // deep copy: b's vec is independent of a's

    push(a.items, 30)        // grows only a
    set(b.items, 0, 99)      // changes only b

    print(len(a.items))      // 3
    print(len(b.items))      // 2
    print(get(a.items, 0))   // 10  — a unaffected by b's set
    print(get(b.items, 0))   // 99
}
