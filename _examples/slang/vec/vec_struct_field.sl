// @test: exit_code=0
// @test: stdout=2\n20\n9\n
// A vec as a struct field: the struct owns the vec and frees it at scope exit
// (the heap stays balanced). vec is a single pointer, so it fits a field like a
// string does.
Bag = struct {
    var items: vec
    val tag: s64
}

main = () {
    var b = Bag{ vec(), 9 }
    push(b.items, 10)
    push(b.items, 20)
    print(len(b.items))    // 2
    print(get(b.items, 1)) // 20
    print(b.tag)           // 9
}
