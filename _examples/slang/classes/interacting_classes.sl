// @test: exit_code=0
// @test: stdout=150\n200\n50\n
// Multiple classes interacting through method calls and ownership

Wallet = class {
    var balance: s64

    create = (initial: s64) -> *Wallet {
        return new Wallet{ initial }
    }

    deposit = (self: &&Wallet, amount: s64) -> bool {
        if amount <= 0 { return false }
        self.balance = self.balance + amount
        return true
    }

    withdraw = (self: &&Wallet, amount: s64) -> bool {
        if amount <= 0 || amount > self.balance { return false }
        self.balance = self.balance - amount
        return true
    }

    get_balance = (self: &Wallet) -> s64 {
        return self.balance
    }
}

transfer = (from: &&Wallet, to: &&Wallet, amount: s64) -> bool {
    if amount <= 0 || from.balance < amount {
        return false
    }
    val withdrew = from.withdraw(amount)
    if !withdrew { return false }
    val deposited = to.deposit(amount)
    return deposited
}

main = () {
    val alice = Wallet.create(100)
    val bob = Wallet.create(200)

    val ok = transfer(bob, alice, 50)
    assert(ok, "transfer should succeed")

    assert(alice.get_balance() == 150, "alice should have 150")
    assert(bob.get_balance() == 150, "bob should have 150")
    print(alice.get_balance())

    val bad = transfer(alice, bob, 500)
    assert(bad == false, "overdraw should fail")
    assert(alice.get_balance() == 150, "alice unchanged")

    transfer(alice, bob, 50)
    transfer(bob, alice, 100)
    assert(alice.get_balance() == 200, "alice should have 200")
    assert(bob.get_balance() == 100, "bob should have 100")
    print(alice.get_balance())

    alice.deposit(300)
    alice.withdraw(450)
    assert(alice.get_balance() == 50, "alice should have 50")
    print(alice.get_balance())
}
