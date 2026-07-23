// @test: exit_code=0
// @test: stdout=9\n4\nt0\nz9\n2\n3\n
// Regression: assigning a struct value into an array element must free the old
// element, deep-copy a borrowed source (independence), and address the slot by
// word stride (elements are stored as pointers). The old code strided by the
// whole struct size (wrong slot at index >= 1), skipped the old-element free
// (leak), and aliased the source (double-free for heap sub-fields).
Plain = struct { var a: s64  var b: s64 }
Named = struct { var name: string }
Bag = struct { var items: vec }

main = () {
    var arr = [Plain{ 1, 2 }, Plain{ 3, 4 }]
    arr[0] = Plain{ 9, 8 }
    print(arr[0].a)      // 9
    print(arr[1].b)      // 4 — index 1 untouched (correct stride)

    var t = Named{ "t${0}" }
    var narr = [Named{ "a${0}" }, Named{ "b${0}" }]
    narr[1] = t          // deep copy into element 1
    t.name = "z${9}"     // mutate the source
    print(narr[1].name)  // t0 — element owns an independent copy
    print(t.name)        // z9

    var barr = [Bag{ vec() }, Bag{ vec() }]
    push(barr[0].items, 10)
    push(barr[0].items, 20)
    barr[1] = barr[0]           // copy element 0 into element 1
    print(len(barr[1].items))   // 2
    push(barr[0].items, 30)     // grows only element 0
    print(len(barr[0].items))   // 3 — element 1 stays at 2 (independent)
}
