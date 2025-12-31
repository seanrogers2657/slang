// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=struct 'Point' is already declared
struct Point(val x: i64)

struct Point(val y: i64)

main = () {}
