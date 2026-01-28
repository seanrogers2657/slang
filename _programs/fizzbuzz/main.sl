// FizzBuzz implementation
main = () {
    var i = 1
    var fizz_count = 0
    var buzz_count = 0
    var fizz_buzz_count = 0

    for ; i <= 100; i = i + 1 {
        if i % 15 == 0 {
            print("FizzBuzz")
            fizz_buzz_count = fizz_buzz_count + 1
        } else if i % 3 == 0 {
            print("Fizz")
            fizz_count = fizz_count + 1
        } else if i % 5 == 0 {
            print("Buzz")
            buzz_count = buzz_count + 1
        } else {
            print(i)
        }
    }

    // Verify counts:
    // FizzBuzz (divisible by 15): 15, 30, 45, 60, 75, 90 = 6
    // Fizz only (divisible by 3 but not 15): 33 - 6 = 27
    // Buzz only (divisible by 5 but not 15): 20 - 6 = 14
    assert(fizz_buzz_count == 6, "should have 6 FizzBuzz")
    assert(fizz_count == 27, "should have 27 Fizz")
    assert(buzz_count == 14, "should have 14 Buzz")
    assert(i == 101, "should complete 100 iterations")

    print("FizzBuzz test passed!")
}
