// @test: stdout=5433\n
// db.connection_port = config.db_port + 1
// If init order were wrong, config.db_port would be 0 and result would be 1
import "db"

main = () {
    print(db.connection_port)
}
