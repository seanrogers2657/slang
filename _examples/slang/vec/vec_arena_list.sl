// @test: exit_code=0
// @test: stdout=10\n20\n30\n60\n
// Arena linked list backed by a GROWABLE vec instead of a fixed array: nodes are
// appended dynamically, links are indices (value*2 = next slot, +1 = stored
// value), and the whole arena frees at scope exit.
main = () {
    var values = vec()       // node value at index i
    var nexts = vec()        // next index at index i (-1 = end)

    // build head(0)=10 -> 1=20 -> 2=30, appended dynamically
    push(values, 10)
    push(nexts, 1)
    push(values, 20)
    push(nexts, 2)
    push(values, 30)
    push(nexts, 0 - 1)

    var total = 0
    var cur = 0
    while cur != 0 - 1 {
        print(get(values, cur))
        total = total + get(values, cur)
        cur = get(nexts, cur)
    }
    print(total)             // 60
}
