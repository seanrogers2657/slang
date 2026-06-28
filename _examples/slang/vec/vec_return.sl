// @test: exit_code=0
// @test: stdout=5\n0\n16\n
// Build a vec inside a function and return it by value: it is copied to the
// caller (like a string), the local is freed, and the caller owns the copy.
squares = (n: s64) -> vec {
    var v = vec()
    var i = 0
    while i < n {
        push(v, i * i)
        i = i + 1
    }
    return v
}

main = () {
    val v = squares(5)
    print(len(v))      // 5
    print(get(v, 0))   // 0
    print(get(v, 4))   // 16
}
