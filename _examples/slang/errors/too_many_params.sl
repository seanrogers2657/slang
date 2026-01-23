// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=maximum allowed is 8
tooMany = (a: s64, b: s64, c: s64, d: s64, e: s64, f: s64, g: s64, h: s64, i: s64) {
    print(a)
}

main = () {
    tooMany(1, 2, 3, 4, 5, 6, 7, 8, 9)
}
