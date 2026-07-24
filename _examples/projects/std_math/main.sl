// @test: stdout=7\n3\n9\n-1\n5\n
// Imports the embedded standard library — no packages/ directory required.
import "std/math"

main = () {
    print(math.abs(-7))       // 7
    print(math.min(3, 9))     // 3
    print(math.max(3, 9))     // 9
    print(math.sign(-4))      // -1
    print(math.clamp(12, 0, 5)) // 5
}
