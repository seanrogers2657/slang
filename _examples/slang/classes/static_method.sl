// @test: exit_code=42
// Static method call on a class

Math = class {
    add = (a: s64, b: s64) -> s64 {
        return a + b
    }
}

main = () {
    val result = Math.add(40, 2)
    exit(result)
}
