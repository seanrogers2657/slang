// @test: exit_code=100
// Singleton object with static methods

Math = object {
    add = (a: i64, b: i64) -> i64 {
        return a + b
    }

    mul = (a: i64, b: i64) -> i64 {
        return a * b
    }

    square = (x: i64) -> i64 {
        return Math.mul(x, x)
    }
}

main = () {
    val sum = Math.add(6, 4)   // 10
    val result = Math.square(sum)  // 100
    exit(result)
}
