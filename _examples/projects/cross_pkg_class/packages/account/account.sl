Account = class {
    var balance: s64

    get_balance = (self: &Account) -> s64 {
        return self.balance
    }

    deposit = (self: &&Account, amount: s64) {
        self.balance = self.balance + amount
    }
}
