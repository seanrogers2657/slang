// @test: exit_code=0
// @test: stdout=2\n1\n
// Regression: a vec field inside a `new`-allocated struct AND class must be
// freed when the owner is freed at scope exit. The deep-free walk has to recurse
// into the aggregate's fields for classes too (not just structs), or the vec
// header/buffer leaks and the runtime aborts on the unbalanced heap.
SBag = struct { var items: vec  val tag: s64 }
CBag = class { var items: vec  val tag: s64 }

main = () {
    val s = new SBag{ vec(), 1 }
    push(s.items, 10)
    push(s.items, 20)
    print(len(s.items))          // 2

    val c = new CBag{ vec(), 9 }
    push(c.items, 7)
    print(len(c.items))          // 1
}                                // both vec fields freed here; heap balanced
