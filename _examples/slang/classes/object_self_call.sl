// @test: exit_code=24
// Test object (singleton) calling its own static methods

Math = object {
    // Simple methods
    double = (x: s64) -> s64 {
        return x + x
    }

    triple = (x: s64) -> s64 {
        return x + x + x
    }

    // Method calling another method in same object
    double_then_triple = (x: s64) -> s64 {
        val doubled = Math.double(x)
        return Math.triple(doubled)
    }
}

main = () {
    // 4 -> double -> 8 -> triple -> 24
    val result = Math.double_then_triple(4)
    exit(result)
}
