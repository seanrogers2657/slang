// FizzBuzz implementation
main = () {
    var i = 1
    var fizzCount = 0
    var buzzCount = 0
    var fizzBuzzCount = 0

    for ; i <= 100; i = i + 1 {
        if i % 15 == 0 {
            print("FizzBuzz")
            fizzBuzzCount = fizzBuzzCount + 1
        } else if i % 3 == 0 {
            print("Fizz")
            fizzCount = fizzCount + 1
        } else if i % 5 == 0 {
            print("Buzz")
            buzzCount = buzzCount + 1
        } else {
            print(i)
        }
    }

    // Verify counts:
    // FizzBuzz (divisible by 15): 15, 30, 45, 60, 75, 90 = 6
    // Fizz only (divisible by 3 but not 15): 33 - 6 = 27
    // Buzz only (divisible by 5 but not 15): 20 - 6 = 14
    assert(fizzBuzzCount == 6, "should have 6 FizzBuzz")
    assert(fizzCount == 27, "should have 27 Fizz")
    assert(buzzCount == 14, "should have 14 Buzz")
    assert(i == 101, "should complete 100 iterations")

    print("FizzBuzz test passed!")
}
