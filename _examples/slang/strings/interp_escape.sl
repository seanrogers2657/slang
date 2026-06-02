// @test: exit_code=0
// @test: stdout=$5.00\ncost $ here\nliteral ${ }\n
// Escaping: \$ is a literal dollar; a $ not followed by { or a letter is literal.
main = () {
    print("\$5.00")
    print("cost $ here")
    print("literal \${ }")
}
