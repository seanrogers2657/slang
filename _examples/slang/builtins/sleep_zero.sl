// @test: exit_code=0
// Test sleep with zero duration (should return immediately)

main = () {
    sleep(0)
    exit(0)
}
