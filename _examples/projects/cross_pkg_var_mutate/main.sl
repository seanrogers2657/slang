// @test: stdout=0\n1\n2\n
import "counter"

main = () {
    print(counter.get_count())
    counter.increment()
    print(counter.get_count())
    counter.increment()
    print(counter.get_count())
}
