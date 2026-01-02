// @test: exit_code=0
// @test: stdout=true\n
// Test returning a non-null value from a function with nullable return type
getValue = () -> i64? {
    return 42
}

main = () {
    val x: i64? = getValue()
    print(x != null)  // true - function returned a value

    // We can't unwrap yet without if-narrowing, but we verified it's not null
    // For now, just test that the tag is correct
    if x != null {
        // TODO: implement narrowing to access the value
    }
}
