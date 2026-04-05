// @test: stdout=8080\n
cfg = import "app/config"

main = () {
    print(cfg.port)
}
