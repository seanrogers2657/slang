// @test: stdout=10\n
import "geometry"

// Function using qualified type in parameter and return type
extract_x = (p: *geometry.Point) -> s64 {
    return p.x
}

main = () {
    val p = new geometry.Point{ 10, 20 }
    print(extract_x(p))
}
