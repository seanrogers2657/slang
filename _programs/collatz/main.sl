// Collatz Conjecture Explorer
// Tests: while loops, complex conditionals, modulo, division,
// when expressions, counters, max tracking, assertions
//
// The Collatz conjecture states that for any positive integer n:
//   - If n is even, divide by 2
//   - If n is odd, multiply by 3 and add 1
//   - Eventually you reach 1

collatz_steps = (start: s64) -> s64 {
    var n = start
    var steps: s64 = 0
    while n != 1 {
        n = when {
            n % 2 == 0 -> n / 2
            else -> n * 3 + 1
        }
        steps = steps + 1
    }
    return steps
}

collatz_max = (start: s64) -> s64 {
    var n = start
    var peak = start
    while n != 1 {
        n = if n % 2 == 0 { n / 2 } else { n * 3 + 1 }
        if n > peak {
            peak = n
        }
    }
    return peak
}

main = () {
    // Known Collatz sequences:
    // 1 -> 0 steps
    // 2 -> 1 step (2 -> 1)
    // 3 -> 7 steps (3->10->5->16->8->4->2->1)
    // 6 -> 8 steps
    // 7 -> 16 steps
    // 27 -> 111 steps (famously long)

    assert(collatz_steps(1) == 0, "collatz(1) should be 0 steps")
    assert(collatz_steps(2) == 1, "collatz(2) should be 1 step")
    assert(collatz_steps(3) == 7, "collatz(3) should be 7 steps")
    assert(collatz_steps(6) == 8, "collatz(6) should be 8 steps")
    assert(collatz_steps(7) == 16, "collatz(7) should be 16 steps")
    assert(collatz_steps(27) == 111, "collatz(27) should be 111 steps")

    print("Collatz step counts verified")

    // Track which starting number 1-50 has the most steps
    var max_steps: s64 = 0
    var max_start: s64 = 0
    for (var i = 1; i <= 50; i = i + 1) {
        val steps = collatz_steps(i)
        if steps > max_steps {
            max_steps = steps
            max_start = i
        }
    }

    // Starting value 27 has the most steps (111) among 1-50
    assert(max_start == 27, "27 should have the most steps in 1-50")
    assert(max_steps == 111, "max steps should be 111")
    print("Longest sequence in 1-50:")
    print(max_start)
    print(max_steps)

    // Check peak values in Collatz sequences
    // 3 peaks at 16, 7 peaks at 52, 27 peaks at 9232
    assert(collatz_max(3) == 16, "collatz(3) should peak at 16")
    assert(collatz_max(7) == 52, "collatz(7) should peak at 52")
    assert(collatz_max(27) == 9232, "collatz(27) should peak at 9232")

    print("Peak values verified")

    // Verify ALL numbers 1-100 eventually reach 1
    // (the while loop terminates for all of them)
    var all_converge = true
    for (var i = 1; i <= 100; i = i + 1) {
        val steps = collatz_steps(i)
        if steps < 0 {
            all_converge = false
        }
    }
    assert(all_converge, "all numbers 1-100 should converge")

    print("Collatz conjecture test passed!")
}
