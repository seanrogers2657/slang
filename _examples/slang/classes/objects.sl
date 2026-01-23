// @test: exit_code=100
// Singleton object with static methods

Math = object {
    add = (a: s64, b: s64) -> s64 {
        return a + b
    }

    mul = (a: s64, b: s64) -> s64 {
        return a * b
    }

    square = (x: s64) -> s64 {
        return Math.mul(x, x)
    }
}

main = () {
    val sum = Math.add(6, 4)   // 10
    val result = Math.square(sum)  // 100
    exit(result)
}
