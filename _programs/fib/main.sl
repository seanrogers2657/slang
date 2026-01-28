// Fibonacci sequence
main = () {
    var a: s64 = 0
    var b: s64 = 1
    var c: s64 = 0
    var i = 0
    for ; i < 50; i = i + 1 {
        // Verify known Fibonacci values
        if i == 0 { assert(a == 0, "fib(0) should be 0") }
        if i == 1 { assert(a == 1, "fib(1) should be 1") }
        if i == 2 { assert(a == 1, "fib(2) should be 1") }
        if i == 10 { assert(a == 55, "fib(10) should be 55") }
        if i == 20 { assert(a == 6765, "fib(20) should be 6765") }

        print(a)
        c = a + b
        a = b
        b = c
    }

    assert(i == 50, "should complete 50 iterations")
    print("Fibonacci test passed!")
}
