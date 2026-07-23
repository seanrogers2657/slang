// @test: exit_code=0
// @test: stdout=6765\n
// Memoization. The cache is a vec threaded through the recursion. A vec arg is
// passed as a handle (not copied at the call boundary), so writes in deep calls
// are visible to shallow ones — the cache really accumulates.
fib = (cache: vec, n: s64) -> s64 {
    if get(cache, n) >= 0 {
        return get(cache, n)
    }
    val r = if n <= 1 { n } else { fib(cache, n - 1) + fib(cache, n - 2) }
    set(cache, n, r)
    return r
}

main = () {
    var cache = vec()
    var i = 0
    while i < 21 {
        push(cache, 0 - 1)   // -1 = not yet computed
        i = i + 1
    }
    print(fib(cache, 20))   // 6765
}
