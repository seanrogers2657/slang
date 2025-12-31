// @test: exit_code=0
// @test: stdout=false\n
// Verify short-circuit: right side should NOT be evaluated when left is false
side_effect = () -> bool {
    print(999)  // This should NOT print if short-circuit works
    return true
}

main = () {
    val result = false && side_effect()
    print(result)  // prints false
}
