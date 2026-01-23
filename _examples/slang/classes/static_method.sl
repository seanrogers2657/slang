// @test: exit_code=42
// Static method call on a class

Math = class {
    add = (a: i64, b: i64) -> i64 {
        return a + b
    }
}

main = () {
    val result = Math.add(40, 2)
    exit(result)
}
