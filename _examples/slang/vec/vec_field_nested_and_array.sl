// @test: exit_code=0
// @test: stdout=1\n2\n2\n
// Regression: the deep-free walk must reach vec fields at every nesting, not
// just directly on a `new`-allocated aggregate:
//   - an embedded aggregate VALUE field that itself holds a vec (its heap fields
//     live inside the owner's allocation and must be freed without freeing the
//     embedded region), and
//   - elements of an array of owned pointers, each pointee holding a vec.
// .copy() of the nested owner must also deep-copy the embedded vec. A miss in
// any of these leaks (or double-frees) and the runtime aborts on the unbalanced
// heap.
Inner = struct { var items: vec }
Outer = struct { var inner: Inner  val tag: s64 }
Bag = struct { var items: vec  val tag: s64 }

main = () {
    // Embedded aggregate value with a vec, heap-allocated, then deep-copied.
    val a = new Outer{ Inner{ vec() }, 1 }
    push(a.inner.items, 5)
    val b = a.copy()
    push(b.inner.items, 6)       // grows only the copy
    print(len(a.inner.items))    // 1  — embedded vec copied independently
    print(len(b.inner.items))    // 2

    // Array of owned pointers, each pointee holding a vec.
    var arr = [new Bag{ vec(), 1 }, new Bag{ vec(), 2 }]
    push(arr[0].items, 9)
    print(arr[1].tag)            // 2  — both pointees' vecs freed at scope exit
}
