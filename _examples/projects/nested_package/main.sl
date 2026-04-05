// @test: stdout=255\n128\n
graphics_color = import "graphics/color"

main = () {
    print(graphics_color.red())
    print(graphics_color.blue())
}
