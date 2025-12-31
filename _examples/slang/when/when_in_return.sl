// @test: exit_code=50
// Tests when expression used directly in return
getValue = (x: i64) -> i64 {
    return when {
        x > 100 -> 100
        x > 10 -> 50
        else -> 0
    }
}

main = () {
    exit(getValue(25))
}
