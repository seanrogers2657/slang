// @test: exit_code=0
// @test: stdout=null\ninner\ninner\nouter\ninner\ninner\nouter\ndone\n
// Regression test: Nullable variable in nested loops
// Tests phi node type inference when nullable variables are used across nested loop boundaries

main = () {
    var head: s64? = null
    var x = 0
    while x < 2 {
        for (var count = 0; count < 2; count = count + 1) {
            if head == null {
                print("null")
            }
            head = count
            print("inner")
        }
        print("outer")
        x = x + 1
    }
    print("done")
}
