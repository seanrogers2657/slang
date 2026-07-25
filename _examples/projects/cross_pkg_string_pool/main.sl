// @test: exit_code=0
// @test: stdout=PKG_ZERO\nPKG_ONE\nPKG_TWO\nROOT_ZERO\nROOT_ONE\n
// Regression test for combining per-package string-constant pools.
//
// Every package numbers its pool from zero, so the root's "ROOT_ZERO" and the
// package's "PKG_ZERO" both want _sl_str0. Merging the IR programs has to remap
// the incoming indices onto a single pool; appending the functions alone leaves
// the root's literals unemitted and its references dangling.
//
// Deliberately uses no feature beyond plain literals so it isolates the pool
// merge itself.
consts = import "consts"

main = () {
    print(consts.zero())
    print(consts.one())
    print(consts.two())
    print("ROOT_ZERO")
    print("ROOT_ONE")
}
