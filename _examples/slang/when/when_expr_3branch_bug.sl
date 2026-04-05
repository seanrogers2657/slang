// @test: exit_code=0
// @test: stdout=10\n2\n
// When expression with bare variable results in val assignment

main = () {
    val a = 10
    val b = 5
    val result = when {
        a > b -> a
        a < b -> b
        else -> 0
    }
    print(result)

    val x = 5
    val y = when {
        x > 10 -> 1
        x > 3 -> 2
        else -> 3
    }
    print(y)
}
