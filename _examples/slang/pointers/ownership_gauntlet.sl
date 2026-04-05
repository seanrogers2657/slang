// @test: exit_code=0
// @test: stdout=55\n110\n
// Complex ownership: create, borrow, mutate, copy, consume chains

Pair = struct {
    var a: s64
    var b: s64
}

make_pair = (a: s64, b: s64) -> *Pair {
    return new Pair{ a, b }
}

sum_pair = (p: &Pair) -> s64 {
    return p.a + p.b
}

double_pair = (p: &&Pair) {
    p.a = p.a * 2
    p.b = p.b * 2
}

swap_pair = (p: &&Pair) {
    val temp = p.a
    p.a = p.b
    p.b = temp
}

consume_and_sum = (p: *Pair) -> s64 {
    return p.a + p.b
}

transform = (p: *Pair) -> *Pair {
    return make_pair(p.a + p.b, p.a * p.b)
}

main = () {
    val original = make_pair(10, 20)

    // Borrow to read
    val s1 = sum_pair(original)
    assert(s1 == 30, "sum should be 30")

    // Borrow to mutate
    swap_pair(original)
    assert(original.a == 20, "after swap a should be 20")
    assert(original.b == 10, "after swap b should be 10")

    // Copy then mutate independently
    val copy1 = original.copy()
    double_pair(copy1)
    assert(copy1.a == 40, "copy doubled a should be 40")
    assert(copy1.b == 20, "copy doubled b should be 20")
    assert(original.a == 20, "original should be unchanged")

    // Transform consumes, produces new
    val transformed = transform(copy1)
    assert(transformed.a == 60, "transformed a should be 60")
    assert(transformed.b == 800, "transformed b should be 800")

    // Accumulate
    var accumulator: s64 = 0
    for (var i = 1; i <= 10; i = i + 1) {
        accumulator = accumulator + i
    }
    assert(accumulator == 55, "accumulator should be 55")
    print(accumulator)

    // make -> double -> sum chain
    var total: s64 = 0
    for (var i = 1; i <= 10; i = i + 1) {
        val p = make_pair(i, i)
        double_pair(p)
        total = total + sum_pair(p)
    }
    assert(total == 220, "total should be 220")

    // Consume the original
    val final_sum = consume_and_sum(original)
    assert(final_sum == 30, "consume should give 30")

    print(total / 2)
}
