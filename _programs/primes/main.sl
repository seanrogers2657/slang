// Prime Number Explorer
// TRIGGERS: Bug 6 (bool ==)

is_prime = (n: s64) -> bool {
    if n < 2 { return false }
    if n < 4 { return true }
    if n % 2 == 0 { return false }
    if n % 3 == 0 { return false }

    var i: s64 = 5
    while i * i <= n {
        if n % i == 0 {
            return false
        }
        if n % (i + 2) == 0 {
            return false
        }
        i = i + 6
    }
    return true
}

count_primes = (limit: s64) -> s64 {
    var count: s64 = 0
    for (var i = 2; i <= limit; i = i + 1) {
        if is_prime(i) {
            count = count + 1
        }
    }
    return count
}

gcd = (a: s64, b: s64) -> s64 {
    var x = a
    var y = b
    while y != 0 {
        val temp = y
        y = x % y
        x = temp
    }
    return x
}

main = () {
    assert(is_prime(2), "2 is prime")
    assert(is_prime(3), "3 is prime")
    assert(is_prime(97), "97 is prime")
    assert(is_prime(7919), "7919 is prime")

    assert(is_prime(0) == false, "0 is not prime")
    assert(is_prime(1) == false, "1 is not prime")
    assert(is_prime(4) == false, "4 is not prime")
    assert(is_prime(100) == false, "100 is not prime")

    assert(count_primes(10) == 4, "pi(10) should be 4")
    assert(count_primes(100) == 25, "pi(100) should be 25")

    print("Prime counts:")
    print(count_primes(10))
    print(count_primes(100))

    // Twin primes up to 100
    var twin_count: s64 = 0
    for (var p = 2; p <= 98; p = p + 1) {
        if is_prime(p) && is_prime(p + 2) {
            twin_count = twin_count + 1
        }
    }
    assert(twin_count == 8, "should have 8 twin prime pairs up to 100")
    print("Twin prime pairs:")
    print(twin_count)

    assert(gcd(12, 8) == 4, "gcd(12,8) should be 4")
    assert(gcd(100, 75) == 25, "gcd(100,75) should be 25")
    assert(gcd(17, 13) == 1, "gcd(17,13) should be 1")

    // Goldbach check: every even number > 2 is sum of two primes (up to 200)
    var goldbach_holds = true
    for (var n = 4; n <= 200; n = n + 2) {
        var found = false
        for (var p = 2; p <= n / 2; p = p + 1) {
            if is_prime(p) && is_prime(n - p) {
                found = true
                break
            }
        }
        if !found {
            goldbach_holds = false
        }
    }
    assert(goldbach_holds, "Goldbach conjecture should hold up to 200")

    print("Prime number test passed!")
}
