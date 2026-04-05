// @test: stdout=5432\n
import "config"

main = () {
    print(config.db_port)
}
