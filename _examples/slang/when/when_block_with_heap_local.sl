// @test: exit_code=0
// @test: stdout=a=4\n10\n
// Regression: a when branch with a block body that declares a heap-owning
// local (here a string) must still yield the block's trailing expression as
// the when-expression's value. The block's scope-cleanup ops used to be
// mistaken for the result (getLastValue), returning garbage instead of 10.
main = () {
    val n = 4
    val x = 5
    val r = when {
        x > 3 -> {
            val t = "a=${n}"
            print(t)
            10
        }
        else -> 20
    }
    print(r)   // 10
}
