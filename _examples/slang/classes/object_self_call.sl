// @test: exit_code=24
// Test object (singleton) calling its own static methods

Math = object {
    // Simple methods
    double = (x: i64) -> i64 {
        return x + x
    }

    triple = (x: i64) -> i64 {
        return x + x + x
    }

    // Method calling another method in same object
    doubleThenTriple = (x: i64) -> i64 {
        val doubled = Math.double(x)
        return Math.triple(doubled)
    }
}

main = () {
    // 4 -> double -> 8 -> triple -> 24
    val result = Math.doubleThenTriple(4)
    exit(result)
}
