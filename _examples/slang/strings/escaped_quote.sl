// @test: exit_code=0
// @test: stdout=a"b\nshe said "hi"\n
// Regression: a string literal containing an escaped double quote (\") must
// compile. The assembler's string scanner used to stop at the escaped quote
// and report an unterminated string.
main = () {
    print("a\"b")
    print("she said \"hi\"")
}
