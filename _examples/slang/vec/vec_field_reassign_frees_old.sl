// @test: exit_code=0
// @test: stdout=1\n3\n
// Regression: reassigning a vec field (a.items = vec()) must free the vec the
// field previously owned, just like a string field does. Otherwise the old
// header+buffer leak and the runtime aborts on the unbalanced heap. Reassigning
// in a loop must free each iteration's vec too.
Bag = struct { var items: vec  val tag: s64 }

main = () {
    val a = new Bag{ vec(), 1 }
    push(a.items, 5)
    push(a.items, 6)

    a.items = vec()          // the old 2-element vec is freed here
    push(a.items, 9)
    print(len(a.items))      // 1

    var i = 0
    while i < 4 {
        a.items = vec()      // each replaced vec is freed
        push(a.items, i)
        i = i + 1
    }
    print(get(a.items, 0))   // 3 — only the final iteration's vec survives
}
