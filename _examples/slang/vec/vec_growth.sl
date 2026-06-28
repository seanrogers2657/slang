// @test: exit_code=0
// @test: stdout=10\n0\n81\n100\n100\n7\n
// Growable vec: pushing past capacity reallocates (doubling), get/set are
// bounds-checked, and copy-on-store (val w = v) makes an independent copy. The
// heap stays balanced (every reallocated buffer and the copy are freed).
main = () {
    var v = vec()
    var i = 0
    while i < 10 {            // grows 0 -> 4 -> 8 -> 16
        push(v, i * i)
        i = i + 1
    }
    print(len(v))            // 10
    print(get(v, 0))         // 0
    print(get(v, 9))         // 81

    set(v, 0, 100)
    print(get(v, 0))         // 100

    val w = v                // independent copy
    set(v, 0, 7)
    print(get(w, 0))         // 100 (copy unchanged)
    print(get(v, 0))         // 7
}
