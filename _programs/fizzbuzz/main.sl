fn main(): void {
    for (var i = 0; i < 100; i = i + 1) {
        when {
            i % 3 == 0 && i % 5 == 0 -> print("FizzBuzz")
            i % 3 == 0 -> print("Fizz")
            i % 5 == 0 -> print("Buzz")
            else -> print(i)
        }
    }
}
