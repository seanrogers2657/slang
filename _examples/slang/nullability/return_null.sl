// @test: exit_code=0
// @test: stdout=false\n
// Test returning null from a function with nullable return type
maybeGetValue = () -> s64? {
    return null
}

main = () {
    val x: s64? = maybeGetValue()
    print(x != null)  // false - function returned null
}
