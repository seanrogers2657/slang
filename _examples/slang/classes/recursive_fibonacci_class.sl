// @test: exit_code=0
// @test: stdout=0\n1\n1\n2\n3\n5\n8\n13\n21\n34\n55\n89\n
// Recursive fibonacci via static class method, printing the sequence
// Tests: recursion, static methods, loops calling recursive functions

Math = class {
    var unused: s64

    fib = (n: s64) -> s64 {
        if n <= 0 { return 0 }
        if n == 1 { return 1 }
        return Math.fib(n - 1) + Math.fib(n - 2)
    }

    // Iterative version for comparison
    fib_iter = (n: s64) -> s64 {
        if n <= 0 { return 0 }
        if n == 1 { return 1 }
        var a: s64 = 0
        var b: s64 = 1
        for (var i = 2; i <= n; i = i + 1) {
            val c = a + b
            a = b
            b = c
        }
        return b
    }
}

main = () {
    // Print fib(0) through fib(11)
    for (var i = 0; i < 12; i = i + 1) {
        print(Math.fib(i))
    }

    // Verify recursive matches iterative for larger values
    for (var i = 0; i <= 20; i = i + 1) {
        assert(Math.fib(i) == Math.fib_iter(i), "recursive should match iterative")
    }
}
