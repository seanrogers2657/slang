// @test: exit_code=0
// @test: stdout=10\n
// Verify short-circuit: right side not evaluated when left is non-null
sideEffect = () -> s64 {
    print(999)
    return 0
}

main = () {
    val x: s64? = 10
    val result = x ?: sideEffect()
    print(result)
}
