// FizzBuzz implementation
main = () {
    var i = 1
    for ; i <= 100; i = i + 1 {
        when {
            i % 15 == 0 -> print("FizzBuzz")
            i % 3 == 0 -> print("Fizz")
            i % 5 == 0 -> print("Buzz")
            else -> print(i)
        }
    }
}
