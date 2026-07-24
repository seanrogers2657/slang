// @test: exit_code=0
// @test: stdout=true\ntrue\n1\n0\n
// Regression: string == / != must free owning string operands (interpolated
// strings, call results) once the comparison has read them; no binding owns
// them, so they leaked — including when the comparison is an if/while condition.
// A plain variable operand is owned by its binding and must NOT be freed (that
// would be a double free / crash), so exit_code=0 guards both directions.
greet = (n: s64) -> string { return "n=$n" }

main = () {
    val n = 42
    val v = "n=42"
    print("x=$n" == "x=42")     // temp == temp
    print(greet(n) == v || v == greet(n))   // call vs var, both directions

    if "x=$n" == "x=42" {       // string temp in an if condition
        print(1)
    }

    var i = 0
    while "a$i" == "a99" {       // string temp in a while condition
        i = i + 1
    }
    print(i)
}
