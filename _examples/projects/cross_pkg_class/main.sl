// @test: stdout=1000\n1500\n
import "account"

main = () {
    val a = new account.Account{ 1000 }
    print(a.get_balance())

    a.deposit(500)
    print(a.get_balance())
}
