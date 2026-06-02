// @test: exit_code=0
// @test: stdout=Hello, World!\nname is Slang\n
// Basic string interpolation: ${expr} brace form and $name shorthand.
main = () {
    val name = "World"
    print("Hello, ${name}!")
    val lang = "Slang"
    print("name is $lang")
}
